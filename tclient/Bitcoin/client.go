package main

/*
   #cgo LDFLAGS: /Users/zhaocong/Developer/Binance/wallet-core/build/libTrustWalletCore.a /Users/zhaocong/Developer/Binance/wallet-core/build/trezor-crypto/libTrezorCrypto.a /Users/zhaocong/Developer/Binance/wallet-core/build/libprotobuf.a -lstdc++
   #include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWBinanceSigner.h"
   #include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWBinanceProto.h"
   #include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWEthereumAddress.h"
   #include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWPublicKey.h"
   #include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWEthereumSigner.h"
*/
import "C"
import (
	"encoding/hex"
	"fmt"
	"os"
	"unsafe"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/ipfs/go-log"
	"github.com/spf13/viper"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/pb/bitcoin"
)

type EthereumRPC struct {
	Jsonrpc string	`json:jsonrpc`
	Id int `json:id`
	Method string `json:method`
	Params []string `json:params`
}

func main() {
	nonce := 1
	home := "/Users/zhaocong/.test" + os.Args[1]
	vault := "default"
	passphrase := "123456789"

	err := common.ReadConfigFromHome(viper.New(), false, home, vault, passphrase)
	if err != nil {
		panic(err)
	}
	common.TssCfg.Home = home
	common.TssCfg.Vault = vault
	common.TssCfg.Password = passphrase
	initLogLevel(common.TssCfg)
	//c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
	ecdsaPubKey, err := common.LoadEcdsaPubkey(home, vault, passphrase)
	if err != nil {
		panic(err)
	}

	btcecPubKey := (*btcec.PublicKey)(ecdsaPubKey)

	addr, err := btcutil.NewAddressPubKey(btcecPubKey.SerializeUncompressed(), &chaincfg.TestNet3Params)
	if err != nil {
		panic(err)
	}
	fmt.Println(addr.EncodeAddress())

	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		panic(err)
	}

	unspent, err := hex.DecodeString("885684c1995b4e05cbd64154828851476b1fd75fcc51f8250e395355acb6be77")
	input := &bitcoin.SigningInput{
		HashType:  1,
		Amount: 55000,
		ByteFee: 10,
		ToAddress: addr.EncodeAddress(),
		ChangeAddress: addr.EncodeAddress(),
		PrivateKey: [][]byte{[]byte{}},	// deliberately be empty as we are tss
		Utxo: []*bitcoin.UnspentTransaction{
			&bitcoin.UnspentTransaction{
				OutPoint:             &bitcoin.OutPoint{
					Hash:                 unspent,
					Index:                0,
					Sequence:             math.MaxUint32,
				},
				Script:               script,
				Amount:               35000,
			},
		},
	}
	fmt.Println(input)

	//serialized, err := proto.Marshal(input)
	//if err != nil {
	//	panic(err)
	//}
	//in := C.TW_Ethereum_Proto_SigningInput(ByteSliceToTWData(serialized))
	//messageBytes := C.TWEthereumSignerMessage(in)
	//message := TWDataToByteSlice(messageBytes)

	//digest := sha3.NewLegacyKeccak256().Sum(message)
	//m := client.HashToInt(digest, tss.EC())
	//sig, err := c.SignImpl(m)
	//if err != nil {
	//	panic(err)
	//}

	//output := C.TWEthereumSignerTransaction(in, ByteSliceToTWData(sig))
	//outputBytes := TWDataToByteSlice(output)
	//hash := sha256.Sum256(outputBytes)
	//txHash := hex.EncodeToString(hash[:])
	//fmt.Println(txHash)
	//reqPayload := EthereumRPC{
	//	Jsonrpc: "2.0",
	//	Id:      1,
	//	Method:  "eth_sendRawTransaction",
	//	Params:  []string{"0x"+hex.EncodeToString(outputBytes)},
	//}
	//jsonPayload, err := json.Marshal(&reqPayload)
	//if err != nil {
	//	panic(err)
	//}
	//req, err := http.NewRequest("POST", "https://binance-rpc.trustwalletapp.com/v1/broadcast?sync=true", bytes.NewReader(txInHex))
	//req, err := http.NewRequest("POST", "https://ropsten.infura.io/v3/a1ebd19437794205a2916e18e61394ef", bytes.NewReader(jsonPayload))
	//req.Header.Set("Content-Type", "application/json")
	//if err != nil {
	//	panic(err)
	//}
	//res, err := http.DefaultClient.Do(req)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(res)
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)
	log.SetLogLevel("common", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
	log.SetLogLevel("stream-upgrader", "error")
}

// Trust-wallet-core integration needed utilities
func ByteSliceToTWData(bytes []byte) unsafe.Pointer {
	cmem := C.CBytes(bytes)
	data := C.TWDataCreateWithBytes((*C.uchar)(cmem), C.ulong(len(bytes)))
	return data
}

func TWDataToByteSlice(raw unsafe.Pointer) []byte {
	size := C.TWDataSize(raw)
	cmem := C.TWDataBytes(raw)
	bytes := C.GoBytes(unsafe.Pointer(cmem), C.int(size))

	return bytes
}

func TWStringToGoString(raw unsafe.Pointer) string {
	size := C.TWStringSize(raw)
	return C.GoStringN(C.TWStringUTF8Bytes(raw), C.int(size))
}