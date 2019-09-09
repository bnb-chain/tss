package main

/*
#cgo LDFLAGS: /Users/zhaocong/Developer/Binance/wallet-core/build/libTrustWalletCore.a /Users/zhaocong/Developer/Binance/wallet-core/build/trezor-crypto/libTrezorCrypto.a /Users/zhaocong/Developer/Binance/wallet-core/build/libprotobuf.a -lstdc++
#include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWBinanceSigner.h"
#include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWBinanceProto.h"
*/
import "C"
import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/ipfs/go-log"
	"github.com/minio/sha256-simd"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/bech32"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/pb"
)

func main() {
	sequence := 7
	from, _ := GetFromBech32("tbnb1pjhqz6pfp7zre7xpj00rmr0ph276rmdsxjyuv2", "tbnb")
	to, _ := GetFromBech32("tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d", "tbnb")
	home := "/Users/zhaocong/.test1"
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
	c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
	pubKey, err := client.LoadPubkey(home, vault)
	if err != nil {
		panic(err)
	}
	input := &pb.SigningInput{
		AccountNumber: 696296,
		ChainId:       "Binance-Chain-Nile",
		Sequence:      int64(sequence),
		OrderOneof: &pb.SigningInput_SendOrder{
			SendOrder: &pb.SendOrder{
				Inputs: []*pb.SendOrder_Input{
					{
						Address: from,
						Coins:   []*pb.SendOrder_Token{{Amount: 1000, Denom: "BNB"}},
					},
				},
				Outputs: []*pb.SendOrder_Output{
					{
						Address: to,
						Coins:   []*pb.SendOrder_Token{{Amount: 1000, Denom: "BNB"}},
					},
				},
			},
		},
	}

	serialized, err := proto.Marshal(input)
	if err != nil {
		panic(err)
	}
	in := C.TW_Binance_Proto_SigningInput(byteSliceToTWData(serialized))
	messageBytes := C.TWBinanceMessage(in)
	message := twDataToByteSlice(messageBytes)

	sig, err := c.Sign(message)
	if err != nil {
		panic(err)
	}

	pubKeyBytes := pubKey.(secp256k1.PubKeySecp256k1)
	output := C.TWBinanceTransaction(in, byteSliceToTWData(pubKeyBytes[:]), byteSliceToTWData(sig))
	outputBytes := twDataToByteSlice(output)
	txInHex := make([]byte, hex.EncodedLen(len(outputBytes)))
	hex.Encode(txInHex, outputBytes)
	hash := sha256.Sum256(outputBytes)
	txHash := hex.EncodeToString(hash[:])
	fmt.Println(txHash)
	//req, err := http.NewRequest("POST", "https://binance-rpc.trustwalletapp.com/v1/broadcast?sync=true", bytes.NewReader(txInHex))
	req, err := http.NewRequest("POST", "https://testnet-dex.binance.org/api/v1/broadcast?sync=true", bytes.NewReader(txInHex))
	req.Header.Set("Content-Type", "text/plain")
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func byteSliceToTWData(bytes []byte) unsafe.Pointer {
	cmem := C.CBytes(bytes)
	data := C.TWDataCreateWithBytes((*C.uchar)(cmem), C.ulong(len(bytes)))
	return data
}

func twDataToByteSlice(raw unsafe.Pointer) []byte {
	size := C.TWDataSize(raw)
	cmem := C.TWDataBytes(raw)
	bytes := C.GoBytes(unsafe.Pointer(cmem), C.int(size))

	return bytes
}

func GetFromBech32(bech32str, prefix string) ([]byte, error) {
	if len(bech32str) == 0 {
		return nil, errors.New("decoding Bech32 address failed: must provide an address")
	}

	hrp, bz, err := bech32.DecodeAndConvert(bech32str)
	if err != nil {
		return nil, err
	}

	if hrp != prefix {
		return nil, fmt.Errorf("invalid Bech32 prefix; expected %s, got %s", prefix, hrp)
	}

	return bz, nil
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
