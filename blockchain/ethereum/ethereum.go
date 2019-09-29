// +build deluxe

package ethereum

/*
   #cgo LDFLAGS: ${SRCDIR}/../../wallet-core/build/libTrustWalletCore.a ${SRCDIR}/../../wallet-core/build/trezor-crypto/libTrezorCrypto.a ${SRCDIR}/../../wallet-core/build/libprotobuf.a -lstdc++
   #cgo CFLAGS: -I${SRCDIR}/../../wallet-core/include/TrustWalletCore/
   #include "TWBinanceSigner.h"
   #include "TWBinanceProto.h"
   #include "TWEthereumAddress.h"
   #include "TWPublicKey.h"
   #include "TWEthereumSigner.h"
*/
import "C"
import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"

	"github.com/binance-chain/tss/blockchain/common"
)

type Network int

const (
	Mainnet Network = iota
	Ropsten
)

var chainId = map[Network]int{
	Mainnet: 1,
	Ropsten: 3}

var accessPoint = map[Network]string{
	Mainnet: "https://mainnet.infura.io/v3/a1ebd19437794205a2916e18e61394ef",
	Ropsten: "https://ropsten.infura.io/v3/a1ebd19437794205a2916e18e61394ef"}

type EthereumRPC struct {
	Jsonrpc string   `json:"jsonrpc"`
	Id      int      `json:"id"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}

type EthereumError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type EthereumResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      int           `json:"id"`
	Error   EthereumError `json:"error"`
}

type EthereumAccountResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

type Ethereum struct {
	Network                Network
	serializedSigningInput []byte
}

//	btcecPubKey := (*btcec.PublicKey)(publicKey)
//	pubkeyBytes := make([]byte, btcec.PubKeyBytesLenUncompressed, btcec.PubKeyBytesLenUncompressed)
//	copy(pubkeyBytes[:], btcecPubKey.SerializeUncompressed())
func (e *Ethereum) GetAddress(publicKey []byte) (string, error) {
	pk := C.TWPublicKeyCreateWithData(common.ByteSliceToTWData(publicKey), C.TWPublicKeyTypeSECP256k1Extended)
	address := C.TWEthereumAddressCreateWithPublicKey(pk)
	addrStr := C.TWEthereumAddressDescription(address)
	addrBytes := common.TWStringToGoString(addrStr)
	return addrBytes, nil
}

// from format: "0xF69e5eb40551020547E09cD400881026173A376e"
// to format: "0xF69e5eb40551020547E09cD400881026173A376e"
func (e *Ethereum) BuildPreImage(amount int64, from, to, demon string) ([]byte, error) {
	nonce, err := e.queryNonce(from)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account information: %v", err)
	}
	gas := big.NewInt(30000) // TODO: figure out how to retrieve gas
	gasLimit := big.NewInt(314150)
	input := &SigningInput{
		ChainId:   math.PaddedBigBytes(big.NewInt(int64(chainId[e.Network])), 32),
		Nonce:     math.PaddedBigBytes(big.NewInt(nonce), 32),
		GasPrice:  math.PaddedBigBytes(gas, 32),
		GasLimit:  math.PaddedBigBytes(gasLimit, 32),
		ToAddress: to,
		Amount:    math.PaddedBigBytes(big.NewInt(amount), 32),
	}

	serialized, err := proto.Marshal(input)
	if err != nil {
		panic(err)
	}
	e.serializedSigningInput = serialized
	in := C.TW_Ethereum_Proto_SigningInput(common.ByteSliceToTWData(serialized))
	messageBytes := C.TWEthereumSignerMessage(in)
	message := common.TWDataToByteSlice(messageBytes)

	return message, nil
}

func (e *Ethereum) BuildTransaction(signature []byte) ([]byte, error) {
	in := C.TW_Ethereum_Proto_SigningInput(common.ByteSliceToTWData(e.serializedSigningInput))
	output := C.TWEthereumSignerTransaction(in, common.ByteSliceToTWData(signature))
	outputBytes := common.TWDataToByteSlice(output)
	return outputBytes, nil
}

func (e *Ethereum) Broadcast(transaction []byte) ([]byte, error) {
	txInHex := "0x" + hex.EncodeToString(transaction)
	fmt.Println(txInHex)
	reqPayload := EthereumRPC{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "eth_sendRawTransaction",
		Params:  []string{txInHex},
	}
	jsonPayload, err := json.Marshal(&reqPayload)
	if err != nil {
		return nil, err
	}
	//req, err := http.NewRequest("POST", "https://binance-rpc.trustwalletapp.com/v1/broadcast?sync=true", bytes.NewReader(txInHex))
	req, err := http.NewRequest("POST", accessPoint[e.Network], bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(payload))

	var jsonResponse EthereumResponse
	err = json.Unmarshal(payload, &jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse ethereum response: %v", err)
	}

	if res.StatusCode == http.StatusOK && jsonResponse.Error.Code == 0 {
		hash := crypto.Keccak256Hash(transaction)
		return hash[:], nil
	} else {
		return nil, fmt.Errorf("failed to broadcast transaction, status: %d, response: %s", res.StatusCode, string(payload))
	}
}

func (e Ethereum) queryNonce(address string) (nonce int64, err error) {
	reqPayload := EthereumRPC{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "eth_getTransactionCount",
		Params:  []string{address, "latest"},
	}
	jsonPayload, err := json.Marshal(&reqPayload)
	if err != nil {
		return 0, err
	}

	res, err := http.Post(accessPoint[e.Network], "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return 0, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	if res.StatusCode == http.StatusOK {
		var accountInfo EthereumAccountResponse
		err := json.Unmarshal(payload, &accountInfo)
		if err != nil {
			return 0, err
		}
		cleaned := strings.Replace(accountInfo.Result, "0x", "", -1)
		return strconv.ParseInt(cleaned, 16, 64)
	} else {
		return 0, fmt.Errorf("failed to fetch account nonce, status: %d, response: %s", res.StatusCode, string(payload))
	}
}
