// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package presigning

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/binance-chain/tss-lib/common"
	zkpenc "github.com/binance-chain/tss-lib/crypto/zkp/enc"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound1(params *tss.Parameters, key *keygen.LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- *PreSignatureData, dump chan<- *LocalDumpPB) tss.Round {
	return &presign1{
		&base{params, key, temp, out, end, dump, make([]bool, len(params.Parties().IDs())), false, 1}}
}

func (round *presign1) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}

	round.number = 1
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	Pi := round.PartyID()
	round.ok[i] = true

	// Fig 7. Round 1. generate ssid #TODO missing run_id & pre_data idx as input
	ssid, err := round.getSSID()
	if err != nil {
		return round.WrapError(err, Pi)
	}

	// Fig 7. Round 1. sample k and gamma
	KShare := common.GetRandomPositiveInt(round.EC().Params().N)
	GammaShare := common.GetRandomPositiveInt(round.EC().Params().N)
	K, KNonce, err := round.key.PaillierSK.EncryptAndReturnRandomness(KShare)
	if err != nil {
		return round.WrapError(fmt.Errorf("paillier encryption failed"), Pi)
	}
	G, GNonce, err := round.key.PaillierSK.EncryptAndReturnRandomness(GammaShare)
	if err != nil {
		return round.WrapError(fmt.Errorf("paillier encryption failed"), Pi)
	}

	// Fig 7. Round 1. create proof enc
	errChs := make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg := sync.WaitGroup{}
	ContextI := append(ssid, big.NewInt(int64(i)).Bytes()...)
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			proof, err := zkpenc.NewProof(ContextI, round.EC(), &round.key.PaillierSK.PublicKey, K, round.key.NTildej[j], round.key.H1j[j], round.key.H2j[j], KShare, KNonce)
			if err != nil {
				errChs <- round.WrapError(fmt.Errorf("ProofEnc failed: %v", err), Pi)
				return
			}

			r1msg := NewPreSignRound1Message(Pj, round.PartyID(), K, G, proof)
			round.out <- r1msg
		}(j, Pj)
	}
	wg.Wait()
	close(errChs)
	for err := range errChs {
		return err
	}

	round.temp.Ssid = ssid
	round.temp.KShare = KShare
	round.temp.GammaShare = GammaShare
	round.temp.G = G
	round.temp.K = K
	round.temp.KNonce = KNonce
	round.temp.GNonce = GNonce

	du := &LocalDump{
		Temp:     round.temp,
		RoundNum: round.number,
		Index:    i,
	}
	duPB := NewLocalDumpPB(du.Index, du.RoundNum, du.Temp)
	round.dump <- duPB

	return nil
}

func (round *presign1) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.R1msgK {
		if round.ok[j] {
			continue
		}
		if msg == nil || round.temp.R1msgG[j] == nil || round.temp.R1msgProof[j] == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *presign1) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*PreSignRound1Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *presign1) NextRound() tss.Round {
	round.started = false
	return &presign2{round}
}

// ----- //

// helper to call into PrepareForSigning()
func (round *presign1) prepare() error {
	i := round.PartyID().Index

	xi := round.key.Xi
	ks := round.key.Ks
	BigXs := round.key.BigXj

	if round.Threshold()+1 > len(ks) {
		return fmt.Errorf("t+1=%d is not satisfied by the key count of %d", round.Threshold()+1, len(ks))
	}
	wi, BigWs := PrepareForSigning(round.Params().EC(), i, len(ks), xi, ks, BigXs)

	round.temp.W = wi
	round.temp.BigWs = BigWs
	return nil
}
