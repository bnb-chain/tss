// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package resharing

import (
	"bytes"
	"errors"
	"math/big"

	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func (round *round2) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 2
	round.started = true
	round.resetOK() // resets both round.oldOK and round.newOK

	if !round.ReSharingParams().IsNewCommittee() {
		round.allOldOK()
		return nil
	}

	Pi := round.PartyID()
	i := Pi.Index

	// check consistency of SSID
	r1msg := round.temp.dgRound1Messages[0].Content().(*DGRound1Message)
	SSID := r1msg.UnmarshalSSID()
	for j, Pj := range round.OldParties().IDs() {
		if j == 0 || j == i {
			continue
		}
		r1msg := round.temp.dgRound1Messages[j].Content().(*DGRound1Message)
		SSIDj := r1msg.UnmarshalSSID()
		if !bytes.Equal(SSID, SSIDj) {
			return round.WrapError(errors.New("ssid mismatch"), Pj)
		}
	}

	// 2. "broadcast" "ACK" members of the OLD committee
	r2msg2 := NewDGRound2Message2(
		round.OldParties().IDs().Exclude(round.PartyID()), round.PartyID())
	round.temp.dgRound2Message2s[i] = r2msg2
	round.out <- r2msg2

	// 1.
	// generate Paillier public key E_i, private key and proof
	// generate safe primes for ZKPs later on
	// compute ntilde, h1, h2 (uses safe primes)
	// use the pre-params if they were provided to the LocalParty constructor
	var preParams *keygen.LocalPreParams
	if round.save.LocalPreParams.Validate() {
		preParams = &round.save.LocalPreParams
	} else {
		var err error
		preParams, err = keygen.GeneratePreParams(round.SafePrimeGenTimeout())
		if err != nil {
			return round.WrapError(errors.New("pre-params generation failed"), Pi)
		}
	}
	round.save.LocalPreParams = *preParams
	round.save.NTildej[i] = preParams.NTildei
	round.save.H1j[i], round.save.H2j[i] = preParams.H1i, preParams.H2i
	round.save.PaillierPKs[i] = &preParams.PaillierSK.PublicKey

	ContextI := append(SSID, big.NewInt(int64(i)).Bytes()...)
	Phi := new(big.Int).Mul(new(big.Int).Lsh(preParams.P, 1), new(big.Int).Lsh(preParams.Q, 1))
	proofPrm, err := zkpprm.NewProof(ContextI, preParams.H1i, preParams.H2i, preParams.NTildei, Phi, preParams.Beta)
	if err != nil {
		return round.WrapError(errors.New("create proofPrm failed"), Pi)
	}
	SP := new(big.Int).Add(new(big.Int).Lsh(preParams.P, 1), big.NewInt(1))
	SQ := new(big.Int).Add(new(big.Int).Lsh(preParams.Q, 1), big.NewInt(1))
	proofFac, err := zkpfac.NewProof(ContextI, round.EC(), preParams.NTildei, preParams.NTildei, preParams.H1i, preParams.H2i, SP, SQ)
	if err != nil {
		return round.WrapError(errors.New("create proofFac failed"), Pi)
	}
	// TODO proofMod?

	r2msg1 := NewDGRound2Message1(
		round.NewParties().IDs().Exclude(round.PartyID()), round.PartyID(),
		&preParams.PaillierSK.PublicKey, proofPrm, preParams.NTildei, preParams.H1i, preParams.H2i, proofFac)
	round.temp.dgRound2Message1s[i] = r2msg1
	round.out <- r2msg1

	round.temp.SSID = SSID

	return nil
}

func (round *round2) CanAccept(msg tss.ParsedMessage) bool {
	if round.ReSharingParams().IsNewCommittee() {
		if _, ok := msg.Content().(*DGRound2Message1); ok {
			return msg.IsBroadcast()
		}
	}
	if round.ReSharingParams().IsOldCommittee() {
		if _, ok := msg.Content().(*DGRound2Message2); ok {
			return msg.IsBroadcast()
		}
	}
	return false
}

func (round *round2) Update() (bool, *tss.Error) {
	if round.ReSharingParams().IsOldCommittee() && round.ReSharingParameters.IsNewCommittee() {
		// accept messages from new -> old committee
		for j, msg1 := range round.temp.dgRound2Message2s {
			if round.newOK[j] {
				continue
			}
			if msg1 == nil || !round.CanAccept(msg1) {
				return false, nil
			}
			// accept message from new -> committee
			msg2 := round.temp.dgRound2Message1s[j]
			if msg2 == nil || !round.CanAccept(msg2) {
				return false, nil
			}
			round.newOK[j] = true
		}
	} else if round.ReSharingParams().IsOldCommittee() {
		// accept messages from new -> old committee
		for j, msg := range round.temp.dgRound2Message2s {
			if round.newOK[j] {
				continue
			}
			if msg == nil || !round.CanAccept(msg) {
				return false, nil
			}
			round.newOK[j] = true
		}
	} else if round.ReSharingParams().IsNewCommittee() {
		// accept messages from new -> new committee
		for j, msg := range round.temp.dgRound2Message1s {
			if round.newOK[j] {
				continue
			}
			if msg == nil || !round.CanAccept(msg) {
				return false, nil
			}
			round.newOK[j] = true
		}
	} else {
		return false, round.WrapError(errors.New("this party is not in the old or the new committee"), round.PartyID())
	}
	if !round.IsOldCommittee() {
		rnd3 := &round3{round}
		return rnd3.Update()
	}
	return true, nil
}

func (round *round2) NextRound() tss.Round {
	round.started = false
	if round.IsOldCommittee() {
		return &round3{round}
	}
	return &round4{&round3{round}}
}
