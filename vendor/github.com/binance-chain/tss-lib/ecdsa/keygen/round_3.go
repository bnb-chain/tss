// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"
	"math/big"
	sync "sync"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/tss"

	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"
	zkpmod "github.com/binance-chain/tss-lib/crypto/zkp/mod"
)

func (round *round3) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 3
	round.started = true
	round.resetOK()

	Pi := round.PartyID()
	i := Pi.Index
	round.ok[i] = true

	// Fig 5. Round 3.1 / Fig 6. Round 3.1
	errChs := make(chan *tss.Error, (len(round.Parties().IDs())-1)*2)
	wg := sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			contextJ := append(round.temp.ssid, big.NewInt(int64(j)).Bytes()...)
			if ok := round.temp.r2msgpfprm[j].Verify(contextJ, round.save.H1j[j], round.save.H2j[j], round.save.NTildej[j]); !ok {
				errChs <- round.WrapError(errors.New("proofPrm verify failed"), Pj)
			}
		}(j, Pj)

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			if round.save.NTildej[j].BitLen() != SafeBitLen*2 {
				errChs <- round.WrapError(errors.New("paillier-blum modulus too small"), Pj)
			}
			proofPrmList := append(round.temp.r2msgpfprm[j].A[:], round.temp.r2msgpfprm[j].Z[:]...)
			listToHash, err := crypto.FlattenECPoints(round.temp.r2msgVss[j])
			if err != nil {
				errChs <- round.WrapError(err, Pj)
			}
			listToHash = append(listToHash, round.save.PaillierPKs[j].N, round.save.NTildej[j], round.save.H1j[j], round.save.H2j[j], round.temp.r2msgAs[j].X(), round.temp.r2msgAs[j].Y(), round.temp.r2msgRids[j], round.temp.r2msgCmtRandomness[j])
			listToHash = append(listToHash, proofPrmList...)
			VjHash := common.SHA512_256i(listToHash...)
			if VjHash.Cmp(round.temp.r1msgVHashs[j]) != 0 {
				errChs <- round.WrapError(errors.New("verify hash failed"), Pj)
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
		return round.WrapError(errors.New("round3: failed stage 3.1"), culprits...)
	}

	// Fig 5. Round 3.2 / Fig 6. Round 3.2 compute round id
	Rid_all := round.temp.rid
	for j := range round.Parties().IDs() {
		if j == i {
			continue
		}
		Rid_all = new(big.Int).Xor(Rid_all, round.temp.r2msgRids[j])
	}
	RidAllBz := append(round.temp.ssid, Rid_all.Bytes()...)
	// Fig 5. Round 3.2 / Fig 6. Round 3.2 proofs
	SP := new(big.Int).Add(new(big.Int).Lsh(round.save.P, 1), one)
	SQ := new(big.Int).Add(new(big.Int).Lsh(round.save.Q, 1), one)

	ContextI := append(RidAllBz, big.NewInt(int64(i)).Bytes()[:]...)
	proofMod, err := zkpmod.NewProof(ContextI, round.save.NTildei, SP, SQ)
	if err != nil {
		return round.WrapError(errors.New("create proofMod failed"), Pi)
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

			proofFac, err := zkpfac.NewProof(ContextI, round.EC(), round.save.NTildei, round.save.NTildej[j], round.save.H1j[j], round.save.H2j[j], SP, SQ)
			if err != nil {
				errChs <- round.WrapError(errors.New("create proofFac failed"), Pi)
			}

			Cij, err := round.save.PaillierPKs[j].Encrypt(round.temp.shares[j].Share)
			if err != nil {
				errChs <- round.WrapError(errors.New("encrypt error"), Pi)
			}

			r3msg := NewKGRound3Message(Pj, round.PartyID(), Cij, proofMod, proofFac)
			round.out <- r3msg
		}(j, Pj)

	}
	wg.Wait()
	close(errChs)
	for err := range errChs {
		return err
	}

	round.temp.RidAllBz = RidAllBz

	return nil
}

func (round *round3) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*KGRound3Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *round3) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.r3msgxij {
		if round.ok[j] {
			continue
		}
		if msg == nil || round.temp.r3msgpffac[j] == nil || round.temp.r3msgpfmod[j] == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *round3) NextRound() tss.Round {
	round.started = false
	return &round4{round}
}
