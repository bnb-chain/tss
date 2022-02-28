// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package presigning

import (
	"errors"
	"math/big"
	sync "sync"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound6(params *tss.Parameters, key *keygen.LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- *PreSignatureData, dump chan<- *LocalDumpPB) tss.Round {
	return &identification1{&presignout{&presign3{&presign2{&presign1{
		&base{params, key, temp, out, end, dump, make([]bool, len(params.Parties().IDs())), false, 6}}}}}}
}

func (round *identification2) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 6
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	round.ok[i] = true
	q := round.EC().Params().N
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q3, q)

	// Fig 7. Output.2
	errChs := make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg := sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		ContextJ := append(round.temp.Ssid, big.NewInt(int64(j)).Bytes()...)

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			proofMul := round.temp.R5msgProofMul[j]
			ok := proofMul.Verify(ContextJ, round.EC(), round.key.PaillierPKs[j], round.temp.R1msgK[j], round.temp.R1msgG[j], round.temp.R5msgH[j])
			if !ok {
				errChs <- round.WrapError(errors.New("round6: proofmul verify failed"), Pj)
				return
			}

			modN2 := common.ModInt(round.key.PaillierPKs[j].NSquare())
			DeltaShareEnc := round.temp.R5msgH[j]
			Q3Enc, err := round.key.PaillierPKs[j].EncryptWithRandomness(q3, new(big.Int).SetBytes(round.temp.Ssid))
			if err != nil {
				errChs <- round.WrapError(err, round.PartyID())
				return
			}
			for k := range round.Parties().IDs() {
				if k == j {
					continue
				}
				var err error
				Djk := round.temp.DeltaMtADs[j]
				if k != i {
					Djk = round.temp.R5msgDjis[k][j]
				}
				DeltaShareEnc, err = round.key.PaillierPKs[j].HomoAdd(DeltaShareEnc, Djk)
				if err != nil {
					errChs <- round.WrapError(err, Pj)
					return
				}
				Fkj := round.temp.R5msgFjis[j][k]
				FinvEnc := modN2.ModInverse(Fkj)
				//BetaEnc := modN2.Mul(round.temp.r5msgQ3Enc[j], FinvEnc)
				BetaEnc := modN2.Mul(Q3Enc, FinvEnc)
				if err != nil {
					errChs <- round.WrapError(err, Pj)
					return
				}
				DeltaShareEnc, err = round.key.PaillierPKs[j].HomoAdd(DeltaShareEnc, BetaEnc)
				if err != nil {
					errChs <- round.WrapError(err, Pj)
					return
				}
			}
			proofDec := round.temp.R5msgProofDec[j]
			ok = proofDec.Verify(ContextJ, round.EC(), round.key.PaillierPKs[j], DeltaShareEnc, round.temp.R3msgDeltaShare[j], round.key.NTildei, round.key.H1i, round.key.H2i)
			if !ok {
				errChs <- round.WrapError(errors.New("round6: proofdec verify failed"), Pj)
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
		return round.WrapError(errors.New("round6: identification verify failed"), culprits...)
	}

	// mark finished
	round.dump <- nil

	return nil
}

func (round *identification2) Update() (bool, *tss.Error) {
	return true, nil
}

func (round *identification2) CanAccept(msg tss.ParsedMessage) bool {
	return true
}

func (round *identification2) NextRound() tss.Round {
	round.started = false
	return nil
}
