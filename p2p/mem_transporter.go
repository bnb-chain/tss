package p2p

import (
	"github.com/binance-chain/tss/common"
	"sync"
)

var once = sync.Once{}
var registeredTransporters map[common.TssClientId]*memTransporter

// in memory transporter used for testing
type memTransporter struct {
	cid       common.TssClientId
	receiveCh chan common.Msg
}

var _ common.Transporter = (*memTransporter)(nil)

func NewMemTransporter(cid common.TssClientId) common.Transporter {
	t := memTransporter{
		cid:       cid,
		receiveCh: make(chan common.Msg, receiveChBufSize),
	}
	once.Do(func() {
		registeredTransporters = make(map[common.TssClientId]*memTransporter, 0)
	})

	registeredTransporters[cid] = &t
	return &t
}

func (t *memTransporter) Broadcast(msg common.Msg) error {
	for cid, peer := range registeredTransporters {
		if cid != t.cid {
			peer.receiveCh <- msg
		}
	}
	return nil
}

func (t *memTransporter) Send(msg common.Msg, to common.TssClientId) error {
	if peer, ok := registeredTransporters[to]; ok {
		peer.receiveCh <- msg
	}
	return nil
}

func (t *memTransporter) ReceiveCh() <-chan common.Msg {
	return t.receiveCh
}

func (t *memTransporter) Close() error {
	return nil
}
