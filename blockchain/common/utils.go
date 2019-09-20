package common

/*
#cgo LDFLAGS: /Users/zhaocong/Developer/Binance/wallet-core/build/libTrustWalletCore.a /Users/zhaocong/Developer/Binance/wallet-core/build/trezor-crypto/libTrezorCrypto.a /Users/zhaocong/Developer/Binance/wallet-core/build/libprotobuf.a -lstdc++
#include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWData.h"
#include "/Users/zhaocong/Developer/Binance/wallet-core/include/TrustWalletCore/TWString.h"
*/
import "C"
import (
	"unsafe"
)

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
