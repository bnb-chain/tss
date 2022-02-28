// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package presigning

import (
	"errors"
	"math/big"
	"sync"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	zkplogstar "github.com/binance-chain/tss-lib/crypto/zkp/logstar"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound3(params *tss.Parameters, key *keygen.LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- *PreSignatureData, dump chan<- *LocalDumpPB) tss.Round {
	return &presign3{&presign2{&presign1{
		&base{params, key, temp, out, end, dump, make([]bool, len(params.Parties().IDs())), false, 3}}}}
}

func (round *presign3) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 3
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	Pi := round.PartyID()
	round.ok[i] = true
	ContextI := append(round.temp.Ssid, big.NewInt(int64(i)).Bytes()...)

	// Fig 7. Round 3.1 verify proofs received and decrypt alpha share of MtA output
	g := crypto.NewECPointNoCurveCheck(round.EC(), round.EC().Params().Gx, round.EC().Params().Gy)
	errChs := make(chan *tss.Error, (len(round.Parties().IDs())-1)*5)
	wg := sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			BigGammaSharej := round.temp.R2msgBigGammaShare[j]
			ContextJ := append(round.temp.Ssid, big.NewInt(int64(j)).Bytes()...)

			DeltaD := round.temp.R2msgDeltaD[j]
			DeltaF := round.temp.R2msgDeltaF[j]
			proofAffgDelta := round.temp.R2msgDeltaProof[j]
			ok := proofAffgDelta.Verify(ContextJ, round.EC(), &round.key.PaillierSK.PublicKey, round.key.PaillierPKs[j], round.key.NTildei, round.key.H1i, round.key.H2i, round.temp.K, DeltaD, DeltaF, BigGammaSharej)
			if !ok {
				errChs <- round.WrapError(errors.New("failed to verify affg delta"), Pj)
				return
			}
			AlphaDelta, err := round.key.PaillierSK.Decrypt(DeltaD)
			if err != nil {
				errChs <- round.WrapError(errors.New("failed to do mta"), Pi)
				return
			}
			round.temp.DeltaShareAlphas[j] = AlphaDelta

			ChiD := round.temp.R2msgChiD[j]
			ChiF := round.temp.R2msgChiF[j]
			proofAffgChi := round.temp.R2msgChiProof[j]
			ok = proofAffgChi.Verify(ContextJ, round.EC(), &round.key.PaillierSK.PublicKey, round.key.PaillierPKs[j], round.key.NTildei, round.key.H1i, round.key.H2i, round.temp.K, ChiD, ChiF, round.temp.BigWs[j])
			if !ok {
				errChs <- round.WrapError(errors.New("failed to verify affg chi"), Pj)
				return
			}
			AlphaChi, err := round.key.PaillierSK.Decrypt(ChiD)
			if err != nil {
				errChs <- round.WrapError(errors.New("failed to do mta"), Pi)
				return
			}
			round.temp.ChiShareAlphas[j] = AlphaChi

			proofLogstar := round.temp.R2msgProofLogstar[j]
			Gj := round.temp.R1msgG[j]
			ok = proofLogstar.Verify(ContextJ, round.EC(), round.key.PaillierPKs[j], Gj, BigGammaSharej, g, round.key.NTildei, round.key.H1i, round.key.H2i)
			if !ok {
				errChs <- round.WrapError(errors.New("failed to verify logstar"), Pj)
				return
			}
		}(j, Pj)
	}
	wg.Wait()
	close(errChs)
	culprits := make([]*tss.PartyID, 0)
	for err := range errChs {
		culprits = append(culprits, err.Culprits()...)
	}
	if len(culprits) > 0 {
		return round.WrapError(errors.New("round3: mta verify failed"), culprits...)
	}

	// Fig 7. Round 3.2 accumulate results from MtA
	BigGamma := round.temp.BigGammaShare
	for j := range round.Parties().IDs() {
		if j == i {
			continue
		}
		BigGammaShare := round.temp.R2msgBigGammaShare[j]
		var err error
		BigGamma, err = BigGamma.Add(BigGammaShare)
		if err != nil {
			return round.WrapError(errors.New("round3: failed to collect BigGamma"))
		}
	}
	BigDeltaShare := BigGamma.ScalarMult(round.temp.KShare)

	modN := common.ModInt(round.EC().Params().N)
	DeltaShare := modN.Mul(round.temp.KShare, round.temp.GammaShare)
	ChiShare := modN.Mul(round.temp.KShare, round.temp.W)
	for j := range round.Parties().IDs() {
		if j == i {
			continue
		}
		DeltaShare = modN.Add(DeltaShare, round.temp.DeltaShareAlphas[j])
		DeltaShare = modN.Add(DeltaShare, round.temp.DeltaShareBetas[j])

		ChiShare = modN.Add(ChiShare, round.temp.ChiShareAlphas[j])
		ChiShare = modN.Add(ChiShare, round.temp.ChiShareBetas[j])
	}

	errChs = make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg = sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			ProofLogstar, err := zkplogstar.NewProof(ContextI, round.EC(), &round.key.PaillierSK.PublicKey, round.temp.K, BigDeltaShare, BigGamma, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j], round.temp.KShare, round.temp.KNonce)
			if err != nil {
				errChs <- round.WrapError(errors.New("proofLogStar generation failed"), Pi)
				return
			}
			r3msg := NewPreSignRound3Message(Pj, round.PartyID(), DeltaShare, BigDeltaShare, ProofLogstar)
			round.out <- r3msg
		}(j, Pj)
	}
	wg.Wait()
	close(errChs)
	for err := range errChs {
		return err
	}

	round.temp.DeltaShare = DeltaShare
	round.temp.ChiShare = ChiShare
	round.temp.BigDeltaShare = BigDeltaShare
	round.temp.BigGamma = BigGamma

	du := &LocalDump{
		Temp:     round.temp,
		RoundNum: round.number,
		Index:    i,
	}
	duPB := NewLocalDumpPB(du.Index, du.RoundNum, du.Temp)
	round.dump <- duPB

	return nil
}

func (round *presign3) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.R3msgDeltaShare {
		if round.ok[j] {
			continue
		}
		if msg == nil || round.temp.R3msgBigDeltaShare[j] == nil || round.temp.R3msgProofLogstar[j] == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *presign3) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*PreSignRound3Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *presign3) NextRound() tss.Round {
	round.started = false
	return &presignout{round}
}
