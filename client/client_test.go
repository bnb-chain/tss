package client_test

import (
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"

	"github.com/ipfs/go-log"

	. "github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

const (
	TestParticipants = 2
	TestThreshold = 1
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

	wg := sync.WaitGroup{}

	homeBase := path.Join(os.TempDir(), "tss", strconv.Itoa(rand.Int()))
	for i := 0; i < TestParticipants; i++ {
		home := path.Join(homeBase, strconv.Itoa(i))
		err := os.MkdirAll(home, 0700)
		if err != nil {
			t.Fatal(err)
		}
		tssConfig := &common.TssConfig{
			Id:        common.TssClientId(strconv.Itoa(i)),
			Moniker:   strconv.Itoa(i),
			Threshold: TestThreshold,
			Parties:   TestParticipants,
			Password:  "1234qwerasdf",
			Home:      home,
			KDFConfig: common.DefaultKDFConfig(),
		}
		client := NewTssClient(tssConfig, KeygenMode, true)
		wg.Add(1)
		go func() {
			client.Start()
			wg.Done()
		}()
	}

	wg.Wait()
	err := os.RemoveAll(homeBase)
	if err != nil {
		t.Fatal(err)
	}
}
