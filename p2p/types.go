package p2p

import (
	"crypto/sha256"
	"fmt"

	"github.com/binance-chain/tss-lib/protob"
	"github.com/binance-chain/tss-lib/tss"
)

// encapsulation of messages that need to be broadcasted
// only send/receive this message on broadcast_sanity_check turn on
type P2pMessageWithHash struct {
	tss.MessageMetadata
	Hash      [sha256.Size]byte
	originMsg tss.ParsedMessage
}

func (m P2pMessageWithHash) GetTo() []*tss.PartyID {
	return m.To
}

func (m P2pMessageWithHash) GetFrom() *tss.PartyID {
	return m.From
}

func (m P2pMessageWithHash) String() string {
	return fmt.Sprintf("[Hash]%s, hash:%x", m.originMsg.String(), m.Hash)
}

// TODO: no need to implement
func (m P2pMessageWithHash) Content() tss.MessageContent {
	return m.originMsg.Content()
}

// TODO: no need to implement
func (m P2pMessageWithHash) ValidateBasic() bool {
	return true
}

func (m P2pMessageWithHash) Type() string {
	return m.originMsg.Type()
}

func (m P2pMessageWithHash) IsBroadcast() bool {
	return m.originMsg.IsBroadcast()
}

func (m P2pMessageWithHash) IsToOldCommittee() bool {
	return m.originMsg.IsToOldCommittee()
}

// Returns the encoded bytes to send over the wire
func (m P2pMessageWithHash) WireBytes() ([]byte, error) {
	return m.originMsg.WireBytes()
}

// Returns the protobuf message struct to send over the wire
func (m P2pMessageWithHash) WireMsg() *protob.Message {
	return m.originMsg.WireMsg()
}
