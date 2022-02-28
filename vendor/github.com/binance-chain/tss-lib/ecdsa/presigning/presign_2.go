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

	"github.com/binance-chain/tss-lib/crypto"
	zkplogstar "github.com/binance-chain/tss-lib/crypto/zkp/logstar"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound2(params *tss.Parameters, key *keygen.LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- *PreSignatureData, dump chan<- *LocalDumpPB) tss.Round {
	return &presign2{&presign1{
		&base{params, key, temp, out, end, dump, make([]bool, len(params.Parties().IDs())), false, 2}}}
}

func (round *presign2) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 2
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	Pi := round.PartyID()
	round.ok[i] = true

	// Fig 7. Round 2.1 verify received proof enc
	errChs := make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg := sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			Kj := round.temp.R1msgK[j]
			proof := round.temp.R1msgProof[j]
			ContextJ := append(round.temp.Ssid, big.NewInt(int64(j)).Bytes()...)
			ok := proof.Verify(ContextJ, round.EC(), round.key.PaillierPKs[j], round.key.NTildei, round.key.H1i, round.key.H2i, Kj)
			if !ok {
				errChs <- round.WrapError(errors.New("round2: proofEnc verify failed"), Pj)
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
		return round.WrapError(errors.New("round2: proofEnc verify failed"), culprits...)
	}

	// Fig 7. Round 2.2 compute MtA and generate proofs
	BigGammaShare := crypto.ScalarBaseMult(round.Params().EC(), round.temp.GammaShare)
	g := crypto.NewECPointNoCurveCheck(round.EC(), round.EC().Params().Gx, round.EC().Params().Gy)
	ContextI := append(round.temp.Ssid, big.NewInt(int64(i)).Bytes()...)
	errChs = make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg = sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			Kj := round.temp.R1msgK[j]

			DeltaMtA, err := NewMtA(ContextI, round.EC(), Kj, round.temp.GammaShare, BigGammaShare, round.key.PaillierPKs[j], &round.key.PaillierSK.PublicKey, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j])
			if err != nil {
				errChs <- round.WrapError(errors.New("MtADelta failed"), Pi)
				return
			}

			ChiMtA, err := NewMtA(ContextI, round.EC(), Kj, round.temp.W, round.temp.BigWs[i], round.key.PaillierPKs[j], &round.key.PaillierSK.PublicKey, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j])
			if err != nil {
				errChs <- round.WrapError(errors.New("MtAChi failed"), Pi)
				return
			}

			ProofLogstar, err := zkplogstar.NewProof(ContextI, round.EC(), &round.key.PaillierSK.PublicKey, round.temp.G, BigGammaShare, g, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j], round.temp.GammaShare, round.temp.GNonce)
			if err != nil {
				errChs <- round.WrapError(errors.New("prooflogstar failed"), Pi)
				return
			}

			r2msg := NewPreSignRound2Message(Pj, round.PartyID(), BigGammaShare, DeltaMtA.Dji, DeltaMtA.Fji, ChiMtA.Dji, ChiMtA.Fji, DeltaMtA.Proofji, ChiMtA.Proofji, ProofLogstar)
			round.out <- r2msg

			round.temp.DeltaShareBetas[j] = DeltaMtA.Beta
			round.temp.ChiShareBetas[j] = ChiMtA.Beta

			if round.NeedsIdentifaction() {
				// record transcript for presign identification 1
				round.temp.DeltaMtAFs[j] = DeltaMtA.Fji
				round.temp.DeltaMtADs[j] = DeltaMtA.Dji
				round.temp.DeltaMtADProofs[j] = DeltaMtA.Proofji

				// record transcript for sign identification 1
				round.temp.ChiMtAFs[j] = ChiMtA.Fji
				round.temp.ChiMtADs[j] = ChiMtA.Dji
				round.temp.ChiMtADProofs[j] = ChiMtA.Proofji
			}
		}(j, Pj)
	}
	wg.Wait()
	close(errChs)
	for err := range errChs {
		return err
	}

	round.temp.BigGammaShare = BigGammaShare

	du := &LocalDump{
		Temp:     round.temp,
		RoundNum: round.number,
		Index:    i,
	}
	duPB := NewLocalDumpPB(du.Index, du.RoundNum, du.Temp)
	round.dump <- duPB

	return nil
}

func (round *presign2) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.R2msgDeltaD {
		if round.ok[j] {
			continue
		}
		if msg == nil || round.temp.R2msgBigGammaShare[j] == nil ||
			round.temp.R2msgChiD[j] == nil || round.temp.R2msgChiF[j] == nil ||
			round.temp.R2msgChiProof[j] == nil || round.temp.R2msgDeltaF[j] == nil ||
			round.temp.R2msgDeltaProof[j] == nil || round.temp.R2msgProofLogstar[j] == nil {

			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *presign2) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*PreSignRound2Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *presign2) NextRound() tss.Round {
	round.started = false
	return &presign3{round}
}
