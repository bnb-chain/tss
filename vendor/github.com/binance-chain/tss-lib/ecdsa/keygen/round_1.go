// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"
	"math/big"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	zkpsch "github.com/binance-chain/tss-lib/crypto/zkp/sch"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound1(params *tss.Parameters, save *LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- LocalPartySaveData) tss.Round {
	return &round1{
		&base{params, save, temp, out, end, make([]bool, len(params.Parties().IDs())), false, 1}}
}

func (round *round1) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 1
	round.started = true
	round.resetOK()

	Pi := round.PartyID()
	i := Pi.Index
	round.ok[i] = true

	// Fig 5. Round 1. private key part
	ui := common.GetRandomPositiveInt(round.EC().Params().N)

	// Fig 5. Round 1. pub key part, vss shares
	ids := round.Parties().IDs().Keys()
	vs, shares, err := vss.Create(round.Params().EC(), round.Threshold(), ui, ids)
	if err != nil {
		return round.WrapError(err, Pi)
	}

	// Fig 5. Round 1. ProofSch first message
	alphai, Ai := zkpsch.NewAlpha(round.EC())

	// Fig 5. Round 1. Session id
	ridBz, err := common.GetRandomBytes(SafeBitLen)
	if err != nil {
		return round.WrapError(err, Pi)
	}
	rid := new(big.Int).SetBytes(ridBz)

	// Fig 6. Round 1. preparams
	var preParams *LocalPreParams
	if round.save.LocalPreParams.Validate() {
		preParams = &round.save.LocalPreParams
	} else {
		preParams, err = GeneratePreParams(round.SafePrimeGenTimeout())
		if err != nil {
			return round.WrapError(errors.New("pre-params generation failed"), Pi)
		}
		round.save.LocalPreParams = *preParams
	}
	// LocalPreParams has no NTildej[], H1j[], H2j[], PaillierPKs[], fill in our own
	round.save.NTildej[i] = preParams.NTildei
	round.save.H1j[i], round.save.H2j[i] = preParams.H1i, preParams.H2i
	round.save.PaillierPKs[i] = &preParams.PaillierSK.PublicKey

	// Fig 6. Round 1. preparams
	Phi := new(big.Int).Mul(new(big.Int).Lsh(round.save.P, 1), new(big.Int).Lsh(round.save.Q, 1))
	ContextI := append(round.temp.ssid, big.NewInt(int64(i)).Bytes()...)
	proofPrm, err := zkpprm.NewProof(ContextI, round.save.H1i, round.save.H2i, round.save.NTildei, Phi, round.save.Beta)
	if err != nil {
		return round.WrapError(errors.New("create proofPrm failed"), Pi)
	}
	proofPrmList := append(proofPrm.A[:], proofPrm.Z[:]...)

	listToHash, err := crypto.FlattenECPoints(vs)
	if err != nil {
		return round.WrapError(err, Pi)
	}
	cmtRandomnessBz, err := common.GetRandomBytes(SafeBitLen)
	if err != nil {
		return round.WrapError(err, Pi)
	}
	cmtRandomness := new(big.Int).SetBytes(cmtRandomnessBz)
	listToHash = append(listToHash, preParams.PaillierSK.PublicKey.N, preParams.NTildei, preParams.H1i, preParams.H2i, Ai.X(), Ai.Y(), rid, cmtRandomness)
	listToHash = append(listToHash, proofPrmList...)
	VHash := common.SHA512_256i(listToHash...)
	{
		msg := NewKGRound1Message(round.PartyID(), VHash)
		round.out <- msg
	}

	round.save.Ks = ids
	round.save.ShareID = ids[i]

	round.temp.proofPrm = proofPrm
	round.temp.alphai = alphai
	round.temp.Ai = Ai
	round.temp.cmtRandomness = cmtRandomness
	round.temp.rid = rid
	round.temp.vs = vs
	round.temp.ui = ui
	round.temp.shares = shares

	return nil
}

func (round *round1) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*KGRound1Message); ok {
		return msg.IsBroadcast()
	}
	return false
}

func (round *round1) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.r1msgVHashs {
		if round.ok[j] {
			continue
		}
		if msg == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *round1) NextRound() tss.Round {
	round.started = false
	return &round2{round}
}
