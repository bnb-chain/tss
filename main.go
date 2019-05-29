package main

import (
	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/server"
)

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("tss-lib", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("srv", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("trans", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.P2PConfig.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("discovery", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("swarm2", cfg.P2PConfig.LogLevel)
}

func main() {
	cfg, err := common.ReadConfig()
	if err != nil {
		panic(err)
	}

	initLogLevel(cfg)

	switch cfg.Mode {
	case "client":
		client.NewTssClient(cfg, false)
		select {}
	case "server":
		server.NewTssBootstrapServer(cfg.P2PConfig)
		select {}
	case "setup":
		client.Setup(cfg)
	}
}
