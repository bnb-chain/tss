package client

import (
	"encoding/gob"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/bgentry/speakeasy"
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

func init() {
	gob.RegisterName("LocalPartySaveData", keygen.LocalPartySaveData{})
	gob.Register(p2p.P2pMessageWithHash{})
}

func NewTssClient(config common.TssConfig, mock bool, done chan<- bool) *TssClient {
	partyID := types.NewPartyID(string(config.Id), config.Moniker)
	unsortedPartyIds := make(types.UnSortedPartyIDs, 0, config.Parties)
	unsortedPartyIds = append(unsortedPartyIds, partyID)
	if !mock {
		for _, peer := range config.P2PConfig.ExpectedPeers {
			unsortedPartyIds = append(unsortedPartyIds,
				types.NewPartyID(
					string(p2p.GetClientIdFromExpecetdPeers(peer)),
					p2p.GetMonikerFromExpectedPeers(peer)))
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
	params := keygen.NewKGParameters(p2pCtx, partyID, config.Parties, config.Threshold)
	// TODO: decide buffer size of this channel
	sendCh := make(chan types.Message, 250)
	saveCh := make(chan keygen.LocalPartySaveData, 250)
	localParty := keygen.NewLocalParty(params, sendCh, saveCh)
	logger.Infof("[%s] initialized localParty: %s", config.Moniker, localParty)
	c := TssClient{
		config:     config,
		localParty: localParty,
	}
	if mock {
		c.transporter = p2p.GetMemTransporter(config.Id)
	} else {
		// will block until peers are connected
		c.transporter = p2p.NewP2PTransporter(config.Home, config.P2PConfig)
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
			logger.Warningf("[%s] Update still waiting for round to finish", tss.config.Moniker)
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
		logger.Infof("[%s] received save data", tss.config.Moniker)

		var passphrase string
		if tss.config.Password != "" {
			passphrase = tss.config.Password
		} else {
			if p, err := speakeasy.Ask("please input password to secure secret key:"); err == nil {
				passphrase = p
			} else {
				panic(err)
			}
		}

		wPriv, err := os.OpenFile(path.Join(tss.config.Home, "sk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			panic(err)
		}
		wPub, err := os.OpenFile(path.Join(tss.config.Home, "pk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			panic(err)
		}
		Save(&msg, tss.transporter.NodeKey(), tss.config.KDFConfig, passphrase, wPriv, wPub)

		if done != nil {
			done <- true
		}
		break
	}
}
