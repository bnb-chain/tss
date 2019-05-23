package common

import (
	"encoding/gob"
	"fmt"

	"github.com/binance-chain/tss-lib/keygen"
)

func init() {
	gob.Register(KGMsg{})
	gob.Register(DummyMsg{})
}

type Msg interface {
	String() string
}

type KGMsg struct {
	msg keygen.KGMessage
	sid string // session id
}

func (m KGMsg) String() string {
	return fmt.Sprintf("sid: %s, type: %s, from: %s, to: %s",
		m.sid,
		m.msg.GetType(),
		m.msg.GetFrom(),
		m.msg.GetTo())
}

// DummyMsg just used for debugging
type DummyMsg struct {
	Content string
}

func (m DummyMsg) String() string {
	return m.Content
}
