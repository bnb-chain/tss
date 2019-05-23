package client

import (
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"

	"time"

	"github.com/binance-chain/tss-lib/keygen"
	"github.com/binance-chain/tss-lib/types"
	"github.com/ipfs/go-log"
)

var logger = log.Logger("tss")

type TssClient struct {
	config      common.TssConfig
	localParty  *keygen.LocalParty
	transporter common.Transporter
}

func NewTssClient(config common.TssConfig) *TssClient {
	params := keygen.NewKGParameters(config.Parties, config.Threshold)
	partyID := types.NewPartyID(string(config.Id), config.Moniker)
	// TODO: decide buffer size of this channel
	sendCh := make(chan types.Message, 250)
	c := TssClient{
		config:      config,
		localParty:  keygen.NewLocalParty(nil, *params, partyID, sendCh),
		transporter: p2p.NewP2PTransporter(config.P2PConfig),
	}

	go c.sendMessageRoutine(sendCh)
	//go c.sendDummyMessageRoutine()
	go c.handleMessageRoutine()

	if _, err := c.localParty.GenerateAndStart(); err != nil {
		panic(err)
	}
	return &c
}

func (tss *TssClient) handleMessageRoutine() {
	for msg := range tss.transporter.ReceiveCh() {
		logger.Info("received message: ", msg)
		tss.localParty.Update(msg)
	}
}

// just used for debugging p2p communication
func (tss *TssClient) sendDummyMessageRoutine() {
	for {
		tss.transporter.Broadcast(common.DummyMsg{string(tss.config.Id)})
		time.Sleep(10 * time.Second)
	}
}

func (tss *TssClient) sendMessageRoutine(sendCh <-chan types.Message) {
	for msg := range sendCh {
		dest := msg.GetTo()
		if dest == nil {
			tss.transporter.Broadcast(msg)
		} else {
			tss.transporter.Send(msg, common.TssClientId(dest.ID))
		}
	}
}
