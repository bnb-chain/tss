package main

import (
	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/server"
	"github.com/ipfs/go-log"
)

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", cfg.LogLevel)
	log.SetLogLevel("discovery", cfg.LogLevel)
	log.SetLogLevel("swarm2", cfg.LogLevel)
}

func main() {
	cfg, err := common.ParseFlags()
	if err != nil {
		panic(err)
	}

	initLogLevel(cfg)

	switch cfg.Mode {
	case "client":
		client.NewTssClient(cfg)
		select {}
	case "server":
		server.NewTssBootstrapServer(cfg.P2PConfig)
		select {}
	}
}
