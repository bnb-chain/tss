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
	sendCh := make(chan keygen.KGMessage, 250)
	c := TssClient{
		config:      config,
		localParty:  keygen.NewLocalParty(nil, *params, *partyID, sendCh),
		transporter: p2p.NewP2PTransporter(config.P2PConfig),
	}

	//go c.sendMessageRoutine(sendCh)
	go c.sendDummyMessageRoutine()
	go c.handleMessageRoutine()
	return &c
}

func (tss *TssClient) handleMessageRoutine() {
	for msg := range tss.transporter.ReceiveCh() {
		logger.Info("received message: ", msg)
	}
}

// just used for debugging
func (tss *TssClient) sendDummyMessageRoutine() {
	for {
		var msg common.Msg
		msg = common.DummyMsg{string(tss.config.Id)}
		tss.transporter.Broadcast(msg)
		time.Sleep(10 * time.Second)
	}
}

func (tss *TssClient) sendMessageRoutine(sendCh <-chan keygen.KGMessage) {
	//for kgMsg := range sendCh {
	//	tss.transporter.Broadcast(common.DummyMsg{string(tss.config.Id)})
	//}
}
