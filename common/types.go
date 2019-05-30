package common

type TssClientId string // Pretty() of peer.ID

func (cid *TssClientId) String() string {
	return string(*cid)
}

func (cl *TssClientId) Set(value string) error {
	*cl = TssClientId(value)
	return nil
}
