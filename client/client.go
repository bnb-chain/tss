package client

import (
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
	"github.com/ipfs/go-log"
	"time"
)

var logger = log.Logger("tss")

type TssClient struct {
	config      common.TssConfig
	transporter common.Transporter
}

func NewTssClient(config common.TssConfig) *TssClient {
	c := TssClient{
		config:      config,
		transporter: p2p.NewP2PTransporter(config.P2PConfig),
	}

	go c.sendMessageRoutine()
	go c.handleMessageRoutine()
	return &c
}

func (tss *TssClient) handleMessageRoutine() {
	for msg := range tss.transporter.ReceiveCh() {
		logger.Info("received message: ", msg)
	}
}

// just used for debugging
func (tss *TssClient) sendMessageRoutine() {
	for {
		tss.transporter.Broadcast(common.DummyMsg{string(tss.config.Id)})
		time.Sleep(10 * time.Second)
	}
}
