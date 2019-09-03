// +build gcc

package main

// #include <string.h>
import "C"
import (
	"unsafe"

	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

//export Sign
func Sign(home, vault, passphrase, message string, pointer unsafe.Pointer) int {
	err := common.ReadConfigFromHome(viper.New(), false, home, vault, passphrase)
	if err != nil {
		return 0
	}
	common.TssCfg.Home = home
	common.TssCfg.Vault = vault
	common.TssCfg.Password = passphrase
	c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
	sig, _ := c.Sign([]byte(message))
	C.memcpy(pointer, C.CBytes(sig), C.ulong(len(sig)))
	return len(sig)
}

func main() {}
