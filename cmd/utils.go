// Most utilitie functions are borrowed from cosmos/cosmos-sdk/client/input.go

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"

	"github.com/binance-chain/tss/common"
)

func getListenAddrs() string {
	addr, err := multiaddr.NewMultiaddr(common.TssCfg.ListenAddr)
	if err != nil {
		panic(err)
	}
	host, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addr))
	if err != nil {
		panic(err)
	}

	builder := strings.Builder{}
	for i, addr := range host.Addrs() {
		if i > 0 {
			fmt.Fprint(&builder, ", ")
		}
		fmt.Fprintf(&builder, "%s", addr)
	}
	host.Close()
	return builder.String()
}
