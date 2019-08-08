package common

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/bgentry/speakeasy"
)

type BootstrapMode uint8

const (
	KeygenMode BootstrapMode = iota
	SignMode
	PreRegroupMode
	RegroupMode
)

// Bootstrapper is helper of pre setting of each kind of client command
// Before keygen, it helps setup peers' moniker and libp2p id, in a "raw" tcp communication way
// For sign, it helps setup signers in libp2p network
// For preregroup, it helps setup new initialized peers' moniker and libp2p id, in a "raw" tcp communication way
// For regroup, it helps setup peers' old and new committee information
type Bootstrapper struct {
	ChannelId       string
	ChannelPassword string
	ExpectedPeers   int
	Cfg             *TssConfig

	Peers sync.Map // id -> peerInfo
}

func NewBootstrapper(expectedPeers int, config *TssConfig) *Bootstrapper {
	// when invoke from anther process (bnbcli), we need set channel id and password here
	if config.ChannelId == "" {
		reader := bufio.NewReader(os.Stdin)
		channelId, err := GetString("please set channel id of this session", reader)
		if err != nil {
			panic(err)
		}
		config.ChannelId = channelId
	}
	if config.ChannelPassword == "" {
		if p, err := speakeasy.Ask("please input password to secure secret bootstrap session:"); err == nil {
			config.ChannelPassword = p
		} else {
			panic(err)
		}
	}
	return &Bootstrapper{
		ChannelId:       config.ChannelId,
		ChannelPassword: config.ChannelPassword,
		ExpectedPeers:   expectedPeers,
		Cfg:             config,
	}
}

func (b *Bootstrapper) HandleBootstrapMsg(peerMsg BootstrapMessage) error {
	if moniker, id, err := Decrypt(peerMsg.PeerInfo, b.ChannelId, b.ChannelPassword); err != nil {
		return err
	} else {
		if info, ok := b.Peers.Load(id); info != nil && ok {
			if moniker != info.(PeerInfo).Moniker {
				return fmt.Errorf("received different moniker for id: %s", id)
			}
		} else {
			pi := PeerInfo{
				Id:         id,
				Moniker:    moniker,
				RemoteAddr: peerMsg.Addr,
				IsOld:      peerMsg.IsOld,
				IsNew:      peerMsg.IsNew,
			}
			b.Peers.Store(id, pi)
		}
	}
	return nil
}

func (b *Bootstrapper) IsFinished() bool {
	switch b.Cfg.BMode {
	case KeygenMode:
		return b.LenOfPeers() == b.ExpectedPeers
	case SignMode:
		return b.LenOfPeers() == b.Cfg.Threshold
	case PreRegroupMode:
		return b.LenOfPeers() == b.ExpectedPeers
	case RegroupMode:
		numOfOld := 0
		numOfNew := 0
		b.Peers.Range(func(_, value interface{}) bool {
			if pi, ok := value.(PeerInfo); ok {
				if pi.IsOld {
					numOfOld++
				}
				if pi.IsNew {
					numOfNew++
				}
			}
			return true
		})
		if TssCfg.IsOldCommittee && TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold && numOfNew+1 >= b.Cfg.NewParties
		} else if TssCfg.IsOldCommittee && !TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold && numOfNew >= b.Cfg.NewParties
		} else if !TssCfg.IsOldCommittee && TssCfg.IsNewCommittee {
			return numOfOld >= b.Cfg.Threshold+1 && numOfNew+1 >= b.Cfg.NewParties
		} else {
			return numOfOld >= b.Cfg.Threshold+1 && numOfNew >= b.Cfg.NewParties
		}
	default:
		return false
	}
}

func (b *Bootstrapper) LenOfPeers() int {
	received := 0
	b.Peers.Range(func(_, _ interface{}) bool {
		received++
		return true
	})
	return received
}

type PeerInfo struct {
	Id         string
	Moniker    string
	RemoteAddr string
	IsOld      bool
	IsNew      bool
}
