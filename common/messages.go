package common

import (
	"encoding/gob"

	"github.com/binance-chain/tss-lib/types"
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
func (m DummyMsg) GetTo() *types.PartyID {
	return nil
}

func (m DummyMsg) GetFrom() *types.PartyID {
	return nil
}

func (m DummyMsg) GetType() string {
	return ""
}
