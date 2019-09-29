// +build deluxe

package binance

/*
#cgo LDFLAGS: ${SRCDIR}/../../wallet-core/build/libTrustWalletCore.a ${SRCDIR}/../../wallet-core/build/trezor-crypto/libTrezorCrypto.a ${SRCDIR}/../../wallet-core/build/libprotobuf.a -lstdc++
#cgo CFLAGS: -I${SRCDIR}/../../wallet-core/include/TrustWalletCore/
#include "TWBinanceSigner.h"
#include "TWBinanceProto.h"
*/
import "C"
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/golang/protobuf/proto"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmBench32 "github.com/tendermint/tendermint/libs/bech32"
	"golang.org/x/crypto/ripemd160"

	"github.com/binance-chain/tss/blockchain/common"
)

type Network int

const (
	Mainnet Network = iota
	Testnet
)

var chainId = map[Network]string{
	Mainnet: "Binance-Chain-Tigris",
	Testnet: "Binance-Chain-Nile"}

var accessPoint = map[Network]string{
	Mainnet: "https://dex.binance.org/api/v1",
	Testnet: "https://testnet-dex.binance.org/api/v1"}

var prefix = map[Network]string{
	Mainnet: "bnb",
	Testnet: "tbnb"}

type Binance struct {
	PublicKey secp256k1.PubKeySecp256k1
	Network   Network

	serializedSigningInput []byte
}

type Balance struct {
	Free   string
	Frozen string
	Locked string
	Symbol string
}

type AccountInfo struct {
	AccountNumber int64      `json:"account_number"`
	Address       string     `json:"address"`
	Balances      []*Balance `json:"balances"`
	Flags         int64      `json:"flags"`
	PublicKey     []uint8    `json:"public_key"`
	Sequence      int64      `json:"sequence"`
}

// btcecPubKey := btcec.PublicKey(*pubKey)
// compressed := btcecPubKey.SerializeCompressed()
func (b *Binance) GetAddress(publicKey []byte) (string, error) {
	hasherSHA256 := sha256.New()
	hasherSHA256.Write(publicKey[:]) // does not error
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error

	address := hasherRIPEMD160.Sum(nil)
	converted, err := bech32.ConvertBits(address, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("encoding bech32 failed: %v", err)
	}
	return bech32.Encode(prefix[b.Network], converted)
}

func (b *Binance) BuildPreImage(amount int64, from, to, demon string) ([]byte, error) {
	accountNumber, sequence, err := b.fetchAccountInfo(from)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account information: %v", err)
	}

	signingInput, err := b.buildPbInput(accountNumber, sequence, amount, from, to, demon)
	if err != nil {
		return nil, fmt.Errorf("failed to build protobuf input: %v", err)
	}

	serialized, err := proto.Marshal(signingInput)
	if err != nil {
		panic(err)
	}
	b.serializedSigningInput = serialized
	in := C.TW_Binance_Proto_SigningInput(common.ByteSliceToTWData(serialized))
	messageBytes := C.TWBinanceSignerMessage(in)
	preImage := crypto.Sha256(common.TWDataToByteSlice(messageBytes))
	return preImage, nil
}

func (b *Binance) BuildTransaction(signature []byte) ([]byte, error) {
	in := C.TW_Binance_Proto_SigningInput(common.ByteSliceToTWData(b.serializedSigningInput))
	output := C.TWBinanceSignerTransaction(in, common.ByteSliceToTWData(b.PublicKey[:]), common.ByteSliceToTWData(signature[:64]))
	outputBytes := common.TWDataToByteSlice(output)
	return outputBytes, nil
}

func (b *Binance) Broadcast(transaction []byte) ([]byte, error) {
	txInHex := make([]byte, hex.EncodedLen(len(transaction)))
	hex.Encode(txInHex, transaction)
	// TODO: integrate with trust-wallet rpc-service
	//req, err := http.NewRequest("POST", "https://binance-rpc.trustwalletapp.com/v1/broadcast?sync=true", bytes.NewReader(txInHex))
	req, err := http.NewRequest("POST", accessPoint[b.Network]+"/broadcast?sync=true", bytes.NewReader(txInHex))
	req.Header.Set("Content-Type", "text/plain")
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusOK {
		hash := sha256.Sum256(transaction)
		return hash[:], nil
	} else {
		payload, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		} else {
			return nil, fmt.Errorf("failed to broadcast transaction, status: %d, response: %s", res.StatusCode, string(payload))
		}
	}
}

func (b Binance) fetchAccountInfo(bech32addr string) (accountNumber, sequence int64, err error) {
	res, err := http.Get(fmt.Sprintf("%s/account/%s", accessPoint[b.Network], bech32addr))
	// TODO: test err is correct
	if err != nil {
		return 0, 0, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, 0, err
	}
	if res.StatusCode == http.StatusOK {
		var accInfo AccountInfo
		err = json.Unmarshal(payload, &accInfo)
		if err != nil {
			return 0, 0, err
		} else {
			return accInfo.AccountNumber, accInfo.Sequence, nil
		}
	} else {
		return 0, 0, fmt.Errorf("failed to fetch account info, status: %d, response: %s", res.StatusCode, string(payload))
	}
}

func (b Binance) buildPbInput(
	accountNumber int64,
	sequence int64,
	amount int64,
	from string,
	to string,
	demon string) (*SigningInput, error) {
	fromAddr, err := b.addressFromBech32(from)
	if err != nil {
		return nil, err
	}

	toAddr, err := b.addressFromBech32(to)
	if err != nil {
		return nil, err
	}

	return &SigningInput{
		AccountNumber: accountNumber,
		ChainId:       chainId[b.Network],
		Sequence:      sequence,
		OrderOneof: &SigningInput_SendOrder{
			SendOrder: &SendOrder{
				Inputs: []*SendOrder_Input{
					{
						Address: fromAddr,
						Coins:   []*SendOrder_Token{{Amount: amount, Denom: demon}},
					},
				},
				Outputs: []*SendOrder_Output{
					{
						Address: toAddr,
						Coins:   []*SendOrder_Token{{Amount: amount, Denom: demon}},
					},
				},
			},
		},
	}, nil
}

func (b Binance) addressFromBech32(bech32addr string) ([]byte, error) {
	if len(bech32addr) == 0 {
		return nil, errors.New("decoding Bech32 address failed: must provide an address")
	}

	hrp, bz, err := tmBench32.DecodeAndConvert(bech32addr)
	if err != nil {
		return nil, err
	}

	if hrp != "tbnb" && hrp != "bnb" {
		return nil, fmt.Errorf("invalid Bech32 prefix: %s; expected tbnb or bnb", bech32addr)
	}

	return bz, nil
}
