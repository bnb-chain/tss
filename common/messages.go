package common

import "fmt"

type Msg interface {
	Bytes() []byte
	String() string
}

type BaseMsg struct {
	sid   string
	round int
	from  TssClientId
}

func (BaseMsg) Bytes() []byte {
	return nil
}

func (m BaseMsg) String() string {
	return fmt.Sprintf("sid: %s, round: %d, from: %s", m.sid, m.round, m.from)
}

// DummyMsg just used for debugging
type DummyMsg struct {
	Content string
}

func (m DummyMsg) Bytes() []byte {
	return []byte(m.Content + "\n")
}

func (m DummyMsg) String() string {
	return m.Content
}
