package p2p

import (
	"github.com/binance-chain/tss-lib/types"

	"github.com/binance-chain/tss/common"
	"sync"
)

var once = sync.Once{}
var registeredTransporters map[common.TssClientId]*memTransporter

// in memory transporter used for testing
type memTransporter struct {
	cid       common.TssClientId
	receiveCh chan types.Message
}

var _ common.Transporter = (*memTransporter)(nil)

func NewMemTransporter(cid common.TssClientId) common.Transporter {
	t := memTransporter{
		cid:       cid,
		receiveCh: make(chan types.Message, receiveChBufSize),
	}
	once.Do(func() {
		registeredTransporters = make(map[common.TssClientId]*memTransporter, 0)
	})

	registeredTransporters[cid] = &t
	return &t
}

func GetMemTransporter(cid common.TssClientId) common.Transporter {
	return registeredTransporters[cid]
}

func (t *memTransporter) NodeKey() []byte {
	return []byte(t.cid.String())
}

func (t *memTransporter) Broadcast(msg types.Message) error {
	logger.Debugf("[%s] Broadcast: %s", t.cid, msg)
	for cid, peer := range registeredTransporters {
		if cid != t.cid {
			peer.receiveCh <- msg
		}
	}
	return nil
}

func (t *memTransporter) Send(msg types.Message, to common.TssClientId) error {
	logger.Debugf("[%s] Sending: %s", t.cid, msg)
	if peer, ok := registeredTransporters[to]; ok {
		peer.receiveCh <- msg
	}
	return nil
}

func (t *memTransporter) ReceiveCh() <-chan types.Message {
	return t.receiveCh
}

func (t *memTransporter) Close() error {
	return nil
}
