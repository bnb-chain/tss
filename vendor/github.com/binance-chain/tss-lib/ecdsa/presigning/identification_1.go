// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package presigning

import (
	"errors"
	"math/big"

	"github.com/binance-chain/tss-lib/common"
	zkpdec "github.com/binance-chain/tss-lib/crypto/zkp/dec"
	zkpmul "github.com/binance-chain/tss-lib/crypto/zkp/mul"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound5(params *tss.Parameters, key *keygen.LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- *PreSignatureData, dump chan<- *LocalDumpPB) tss.Round {
	return &identification1{&presignout{&presign3{&presign2{&presign1{
		&base{params, key, temp, out, end, dump, make([]bool, len(params.Parties().IDs())), false, 5}}}}}}
}

func (round *identification1) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 5
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	Pi := round.PartyID()
	round.ok[i] = true
	ContextI := append(round.temp.Ssid, big.NewInt(int64(i)).Bytes()...)

	// Fig 7. Output.2
	H, rho, err := round.key.PaillierSK.HomoMultObfuscate(round.temp.KShare, round.temp.G)
	if err != nil {
		return round.WrapError(err, Pi)
	}
	proofH, err := zkpmul.NewProof(ContextI, round.EC(), &round.key.PaillierSK.PublicKey, round.temp.K, round.temp.G, H, round.temp.KShare, rho, round.temp.KNonce)
	if err != nil {
		return round.WrapError(err, Pi)
	}

	// Calc DeltaShare2 s.t. Enc(DeltaShare2) = DeltaShareEnc
	DeltaShare2 := new(big.Int).Mul(round.temp.KShare, round.temp.GammaShare)
	for j := range round.Parties().IDs() {
		if j == i {
			continue
		}
		DeltaShare2 = new(big.Int).Add(DeltaShare2, round.temp.DeltaShareAlphas[j])
		DeltaShare2 = new(big.Int).Add(DeltaShare2, round.temp.DeltaShareBetas[j])
	}
	DeltaShareEnc := H
	modN2 := common.ModInt(round.key.PaillierSK.NSquare())
	q := round.EC().Params().N
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q3, q)
	Q3Enc, err := round.key.PaillierSK.EncryptWithRandomness(q3, new(big.Int).SetBytes(round.temp.Ssid))
	if err != nil {
		return round.WrapError(err, Pi)
	}
	if err != nil {
		return round.WrapError(err, Pi)
	}
	for k := range round.Parties().IDs() {
		if k == i {
			continue
		}
		var err error
		DeltaShareEnc, err = round.key.PaillierSK.HomoAdd(DeltaShareEnc, round.temp.R2msgDeltaD[k])
		if err != nil {
			return round.WrapError(err, Pi)
		}
		FinvEnc := modN2.ModInverse(round.temp.DeltaMtAFs[k])
		BetaEnc := modN2.Mul(Q3Enc, FinvEnc)
		if err != nil {
			return round.WrapError(err, Pi)
		}
		DeltaShareEnc, err = round.key.PaillierSK.HomoAdd(DeltaShareEnc, BetaEnc)
		if err != nil {
			return round.WrapError(err, Pi)
		}
	}
	nonce, err := round.key.PaillierSK.GetRandomness(DeltaShareEnc)
	if err != nil {
		return round.WrapError(err, Pi)
	}

	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		proofDec, err := zkpdec.NewProof(ContextI, round.EC(), &round.key.PaillierSK.PublicKey, DeltaShareEnc, round.temp.DeltaShare, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j], DeltaShare2, nonce)
		if err != nil {
			return round.WrapError(err, Pi)
		}

		r6msg := NewIdentificationRound1Message(Pj, round.PartyID(), H, proofH, round.temp.DeltaMtADs, round.temp.DeltaMtAFs, round.temp.DeltaMtADProofs, proofDec)
		round.out <- r6msg
	}

	return nil
}

func (round *identification1) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.R5msgH {
		if round.ok[j] {
			continue
		}
		if msg == nil || round.temp.R5msgDjis[j] == nil || round.temp.R5msgFjis[j] == nil ||
			round.temp.R5msgProofDec[j] == nil || round.temp.R5msgProofMul[j] == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *identification1) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*IdentificationRound1Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *identification1) NextRound() tss.Round {
	round.started = false
	return &identification2{round}
}
