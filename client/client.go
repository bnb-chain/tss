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
	unsortedPartyIds := make(types.UnSortedPartyIDs, 0, config.Parties)
	unsortedPartyIds = append(unsortedPartyIds, partyID)
	// TODO: put other's moniker into config fire
	for _, peer := range config.P2PConfig.ExpectedPeers {
		unsortedPartyIds = append(unsortedPartyIds, types.NewPartyID(string(peer), ""))
	}
	sortedIds := types.SortPartyIDs(unsortedPartyIds)
	p2pCtx := types.NewPeerContext(sortedIds)
	// TODO: decide buffer size of this channel
	sendCh := make(chan types.Message, 250)
	saveCh := make(chan keygen.LocalPartySaveData, 250)
	localParty := keygen.NewLocalParty(p2pCtx, *params, partyID, sendCh, saveCh)
	logger.Infof("initialized localParty: ", localParty)
	c := TssClient{
		config:      config,
		localParty:  localParty,
		transporter: p2p.NewP2PTransporter(config.P2PConfig),
	}

	// has to start local party before network routines in case 2 other peers msg comes before self fully initialized
	if err := c.localParty.StartKeygenRound1(); err != nil {
		panic(err)
	}

	go c.sendMessageRoutine(sendCh)
	//go c.sendDummyMessageRoutine()
	go c.handleMessageRoutine()

	return &c
}

func (tss *TssClient) handleMessageRoutine() {
	for msg := range tss.transporter.ReceiveCh() {
		logger.Info("received message: ", msg)
		_, err := tss.localParty.Update(msg)
		if err != nil {
			logger.Error("error updating local party state: ", err)
		}
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
