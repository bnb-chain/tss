package client

import (
	"strconv"
	"time"

	"github.com/binance-chain/tss-lib/keygen"
	"github.com/binance-chain/tss-lib/types"
	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

var logger = log.Logger("tss")

type TssClient struct {
	config      common.TssConfig
	localParty  *keygen.LocalParty
	transporter common.Transporter
}

func NewTssClient(config common.TssConfig, mock bool, done chan<- bool) *TssClient {
	params := keygen.NewKGParameters(config.Parties, config.Threshold)
	partyID := types.NewPartyID(string(config.Id), config.Moniker)
	unsortedPartyIds := make(types.UnSortedPartyIDs, 0, config.Parties)
	unsortedPartyIds = append(unsortedPartyIds, partyID)
	if !mock {
		// TODO: put other's moniker into config fire
		for _, peer := range config.P2PConfig.ExpectedPeers {
			unsortedPartyIds = append(unsortedPartyIds, types.NewPartyID(string(peer), ""))
		}
	} else {
		for i := 0; i < config.Parties; i++ {
			id, _ := strconv.Atoi(string(config.Id))
			if i != id {
				unsortedPartyIds = append(unsortedPartyIds, types.NewPartyID(strconv.Itoa(i), strconv.Itoa(i)))
			}
		}
	}
	sortedIds := types.SortPartyIDs(unsortedPartyIds)
	p2pCtx := types.NewPeerContext(sortedIds)
	// TODO: decide buffer size of this channel
	sendCh := make(chan types.Message, 250)
	saveCh := make(chan keygen.LocalPartySaveData, 250)
	localParty := keygen.NewLocalParty(p2pCtx, *params, partyID, sendCh, saveCh)
	logger.Infof("[%s] initialized localParty: %s", config.Moniker, localParty)
	c := TssClient{
		config:     config,
		localParty: localParty,
	}
	if mock {
		c.transporter = p2p.GetMemTransporter(config.Id)
	} else {
		// will block until peers are connected
		c.transporter = p2p.NewP2PTransporter(config.P2PConfig)
	}

	// has to start local party before network routines in case 2 other peers msg comes before self fully initialized
	if err := c.localParty.StartKeygenRound1(); err != nil {
		panic(err)
	}

	go c.sendMessageRoutine(sendCh)
	go c.saveDataRoutine(saveCh, done)
	//go c.sendDummyMessageRoutine()
	go c.handleMessageRoutine()

	return &c
}

func (tss *TssClient) handleMessageRoutine() {
	for msg := range tss.transporter.ReceiveCh() {
		logger.Infof("[%s] received message: %s", tss.config.Moniker, msg)
		ok, err := tss.localParty.Update(msg)
		if err != nil {
			logger.Errorf("[%s] error updating local party state: %v", tss.config.Moniker, err)
		} else if !ok {
			logger.Warningf("[%s] failed Update", tss.config.Moniker)
		} else {
			logger.Debugf("[%s] update success", tss.config.Moniker)
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

func (tss *TssClient) saveDataRoutine(saveCh <-chan keygen.LocalPartySaveData, done chan<- bool) {
	for msg := range saveCh {
		logger.Infof("[%s] received save data: %v", tss.config.Moniker, msg)
		if done != nil {
			done <- true
		}
		break
	}
}
