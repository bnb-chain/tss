package common

import (
	"github.com/binance-chain/tss-lib/types"
)

// Transportation layer of TssClient provide Broadcast and Send method over p2p network
// ReceiveCh() provides msgs this client received
// TODO: consider a ControlCh() to expose ready&err msgs to application?
type Transporter interface {
	Broadcast(msg types.Message) error
	Send(msg types.Message, to TssClientId) error
	ReceiveCh() <-chan types.Message // messages have received !consumer of this channel should not taking too long!
	Close() error
}
