// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package signing

import (
	"errors"
	"math/big"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/ecdsa/presigning"
	"github.com/binance-chain/tss-lib/tss"
)

const (
	TaskName = "signing"
)

type (
	base struct {
		*tss.Parameters
		key     *keygen.LocalPartySaveData
		predata *presigning.PreSignatureData
		data    *common.SignatureData
		temp    *localTempData
		out     chan<- tss.Message
		end     chan<- common.SignatureData
		dump    chan<- *LocalDumpPB
		ok      []bool // `ok` tracks parties which have been verified by Update()
		started bool
		number  int
	}
	sign1 struct {
		*base
	}
	signout struct {
		*sign1
	}

	// identification rounds
	identification1 struct {
		*signout
	}
	identification2 struct {
		*identification1
	}
)

var (
	_ tss.Round = (*sign1)(nil)
	_ tss.Round = (*signout)(nil)
	_ tss.Round = (*identification1)(nil)
	_ tss.Round = (*identification2)(nil)
)

// ----- //
func (round *base) SetStarted(status bool) {
	round.started = status
	round.resetOK()

	i := round.PartyID().Index
	round.ok[i] = true
}

func (round *base) Params() *tss.Parameters {
	return round.Parameters
}

func (round *base) RoundNumber() int {
	return round.number
}

// CanProceed is inherited by other rounds
func (round *base) CanProceed() bool {
	if !round.started {
		return false
	}
	for _, ok := range round.ok {
		if !ok {
			return false
		}
	}
	return true
}

// WaitingFor is called by a Party for reporting back to the caller
func (round *base) WaitingFor() []*tss.PartyID {
	Ps := round.Parties().IDs()
	ids := make([]*tss.PartyID, 0, len(round.ok))
	for j, ok := range round.ok {
		if ok {
			continue
		}
		ids = append(ids, Ps[j])
	}
	return ids
}

func (round *base) WrapError(err error, culprits ...*tss.PartyID) *tss.Error {
	return tss.NewError(err, TaskName, round.number, round.PartyID(), culprits...)
}

// ----- //

// `ok` tracks parties which have been verified by Update()
func (round *base) resetOK() {
	for j := range round.ok {
		round.ok[j] = false
	}
}

// get ssid from local params
func (round *base) getSSID() ([]byte, error) {
	ssidList := []*big.Int{round.EC().Params().P, round.EC().Params().N, round.EC().Params().B, round.EC().Params().Gx, round.EC().Params().Gy} // ec curve
	ssidList = append(ssidList, round.Parties().IDs().Keys()...)                                                                                // parties
	BigXjList, err := crypto.FlattenECPoints(round.key.BigXj)
	if err != nil {
		return nil, round.WrapError(errors.New("read BigXj failed"), round.PartyID())
	}
	ssidList = append(ssidList, BigXjList...)         // BigXj
	ssidList = append(ssidList, round.key.NTildej...) // NTilde
	ssidList = append(ssidList, round.key.H1j...)     // h1
	ssidList = append(ssidList, round.key.H2j...)     // h2
	ssid := common.SHA512_256i(ssidList...).Bytes()

	return ssid, nil
}
