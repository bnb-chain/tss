package client

import (
	"strconv"
	"testing"
	"time"

	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

const (
	TestParticipants = 10
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

	for i := 0; i < TestParticipants; i++ {
		p2p.NewMemTransporter(common.TssClientId(strconv.Itoa(i)))
	}

	start := time.Now()
	doneCh := make(chan bool, TestParticipants)

	for i := 0; i < TestParticipants; i++ {
		tssConfig := common.TssConfig{
			Id:        common.TssClientId(strconv.Itoa(i)),
			Moniker:   strconv.Itoa(i),
			Threshold: TestParticipants / 2,
			Parties:   TestParticipants,
			Mode:      "client",
		}
		NewTssClient(tssConfig, true, doneCh)
	}

	i := 0
	for range doneCh {
		logger.Debugf("party i: keygen complete. took %s\n", time.Since(start))
		i++
		if i == TestParticipants {
			break
		}
	}
}
