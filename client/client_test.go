package client

import (
	"strconv"
	"testing"

	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

func initlog() {
	log.SetLogLevel("tss", "debug")
	log.SetLogLevel("tss-lib", "debug")
	log.SetLogLevel("srv", "debug")
	log.SetLogLevel("trans", "debug")
	log.SetLogLevel("p2p_utils", "debug")

	// libp2p loggers
	log.SetLogLevel("dht", "debug")
	log.SetLogLevel("discovery", "debug")
	log.SetLogLevel("swarm2", "debug")
}

func TestWhole(t *testing.T) {
	initlog()

	for i := 0; i < 3; i++ {
		p2p.NewMemTransporter(common.TssClientId(strconv.Itoa(i)))
	}

	for i := 0; i < 3; i++ {
		tssConfig := common.TssConfig{
			Id:        common.TssClientId(strconv.Itoa(i)),
			Moniker:   strconv.Itoa(i),
			Threshold: 2,
			Parties:   3,
			Mode:      "client",
		}
		NewTssClient(tssConfig, true)
	}
	select {}
}
