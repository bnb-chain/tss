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
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
}

func main() {
	cfg, err := common.ReadConfig()
	if err != nil {
		panic(err)
	}

	initLogLevel(cfg)

	switch cfg.Mode {
	case "keygen":
		done := make(chan bool)
		client.NewTssClient(cfg, false, done)
		<-done
	case "sign":
		done := make(chan bool)
		client.NewTssClient(cfg, false, done)
		<-done
	case "server":
		server.NewTssBootstrapServer(cfg.Home, cfg.P2PConfig)
		select {}
	case "setup":
		client.Setup(cfg)
	}
}
