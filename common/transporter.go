package common

// Transportation layer of TssClient provide Broadcast and Send method over p2p network
// ReceiveCh() provides msgs this client received
// TODO: consider a ControlCh() to expose ready&err msgs to application?
type Transporter interface {
	Broadcast(msg Msg) error
	Send(msg Msg, to TssClientId) error
	ReceiveCh() <-chan Msg // messages have received !consumer of this channel should not taking too long!
	Close() error
}
