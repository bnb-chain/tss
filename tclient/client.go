package main

/*
#cgo LDFLAGS: /Users/zhaocong/Developer/Binance/wallet-core/build/libTrustWalletCore.a /Users/zhaocong/Developer/Binance/wallet-core/build/trezor-crypto/libTrezorCrypto.a /Users/zhaocong/Developer/Binance/wallet-core/build/libprotobuf.a /Users/zhaocong/Developer/Binance/tss/trust/libtss.a -lstdc++ -framework CoreFoundation -framework Security
#include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWBinanceSigner.h"
*/
import "C"
import (
	"github.com/gogo/protobuf/proto"

	"github.com/binance-chain/tss/pb"
)

func main() {
	input := &pb.SigningInput{
		AccountNumber: 696296,
		ChainId:       "Binance-Chain-Nile",
		Sequence:      1,
		OrderOneof: &pb.SigningInput_SendOrder{
			SendOrder: &pb.SendOrder{
				Inputs: []*pb.SendOrder_Input{
					{
						Address: []byte("tbnb1pjhqz6pfp7zre7xpj00rmr0ph276rmdsxjyuv2"),
						Coins:   []*pb.SendOrder_Token{{Amount: 1000, Denom: "BNB"}},
					},
				},
				Outputs: []*pb.SendOrder_Output{
					{
						Address: []byte("tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d"),
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
	cmem := C.CBytes(serialized)
	data := C.TWDataCreateWithBytes(cmem, len(serialized))
	C.TWBinanceSignerSign(data)
}
