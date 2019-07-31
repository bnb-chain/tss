package common

import (
	"encoding/gob"

	"github.com/binance-chain/tss-lib/tss"
)

func init() {
	gob.Register(DummyMsg{})
}

type DummyMsg struct {
	Content string
}

func (m DummyMsg) String() string {
	return m.Content
}

// always broadcast
func (m DummyMsg) GetTo() *tss.PartyID {
	return nil
}

func (m DummyMsg) GetFrom() *tss.PartyID {
	return nil
}

func (m DummyMsg) GetType() string {
	return ""
}

func (m DummyMsg) ValidateBasic() bool {
	return true
}

type BootstrapMessage struct {
	ChannelId string // channel id + epoch timestamp in dex
	PeerInfo  []byte // encrypted channelId+moniker+libp2pid
	Addr      string
	IsOld     bool
	IsNew     bool
}

func NewBootstrapMessage(channelId, passphrase, moniker string, id TssClientId, addr string, isOld, isNew bool) (*BootstrapMessage, error) {
	pi, err := Encrypt(passphrase, channelId, moniker, string(id))
	if err != nil {
		return nil, err
	}
	return &BootstrapMessage{
		ChannelId: channelId,
		PeerInfo:  pi,
		Addr:      addr,
		IsOld:     isOld,
		IsNew:     isNew,
	}, nil
}
