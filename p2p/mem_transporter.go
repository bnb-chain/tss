package p2p

import (
	"sync"

	"github.com/binance-chain/tss-lib/tss"

	"github.com/binance-chain/tss/common"
)

var once = sync.Once{}
var registeredTransporters map[common.TssClientId]*memTransporter

// in memory transporter used for testing
type memTransporter struct {
	cid       common.TssClientId
	receiveCh chan tss.Message
}

var _ common.Transporter = (*memTransporter)(nil)

func NewMemTransporter(cid common.TssClientId) common.Transporter {
	t := memTransporter{
		cid:       cid,
		receiveCh: make(chan tss.Message, receiveChBufSize),
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

func (t *memTransporter) Broadcast(msg tss.Message) error {
	logger.Debugf("[%s] Broadcast: %s", t.cid, msg)
	for cid, peer := range registeredTransporters {
		if cid != t.cid {
			peer.receiveCh <- msg
		}
	}
	return nil
}

func (t *memTransporter) Send(msg tss.Message, to common.TssClientId) error {
	logger.Debugf("[%s] Sending: %s", t.cid, msg)
	if peer, ok := registeredTransporters[to]; ok {
		peer.receiveCh <- msg
	}
	return nil
}

func (t *memTransporter) ReceiveCh() <-chan tss.Message {
	return t.receiveCh
}

func (t *memTransporter) Close() error {
	return nil
}
