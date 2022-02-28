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

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	zkpaffg "github.com/binance-chain/tss-lib/crypto/zkp/affg"
	zkpdec "github.com/binance-chain/tss-lib/crypto/zkp/dec"
	zkpenc "github.com/binance-chain/tss-lib/crypto/zkp/enc"
	zkplogstar "github.com/binance-chain/tss-lib/crypto/zkp/logstar"
	zkpmul "github.com/binance-chain/tss-lib/crypto/zkp/mul"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

// Implements Party
// Implements Stringer
var _ tss.Party = (*LocalParty)(nil)
var _ fmt.Stringer = (*LocalParty)(nil)

type (
	LocalParty struct {
		*tss.BaseParty
		params *tss.Parameters

		keys keygen.LocalPartySaveData
		temp localTempData

		// outbound messaging
		out         chan<- tss.Message
		end         chan<- *PreSignatureData
		dump        chan<- *LocalDumpPB
		startRndNum int
	}

	localTempData struct {
		// temp data (thrown away after sign) / round 1
		Ssid   []byte
		W      *big.Int
		BigWs  []*crypto.ECPoint
		KShare *big.Int

		BigGammaShare *crypto.ECPoint
		K             *big.Int
		G             *big.Int
		KNonce        *big.Int
		GNonce        *big.Int
		// round 2
		GammaShare      *big.Int
		DeltaShareBetas []*big.Int
		ChiShareBetas   []*big.Int
		// round 3
		BigGamma         *crypto.ECPoint
		DeltaShareAlphas []*big.Int
		ChiShareAlphas   []*big.Int
		DeltaShare       *big.Int
		ChiShare         *big.Int
		BigDeltaShare    *crypto.ECPoint
		// round 4
		BigR       *crypto.ECPoint
		Rx         *big.Int
		SigmaShare *big.Int
		// msg store
		R1msgG     []*big.Int
		R1msgK     []*big.Int
		R1msgProof []*zkpenc.ProofEnc

		R2msgBigGammaShare []*crypto.ECPoint
		R2msgDeltaD        []*big.Int
		R2msgDeltaF        []*big.Int
		R2msgDeltaProof    []*zkpaffg.ProofAffg
		R2msgChiD          []*big.Int
		R2msgChiF          []*big.Int
		R2msgChiProof      []*zkpaffg.ProofAffg
		R2msgProofLogstar  []*zkplogstar.ProofLogstar

		R3msgDeltaShare    []*big.Int
		R3msgBigDeltaShare []*crypto.ECPoint
		R3msgProofLogstar  []*zkplogstar.ProofLogstar

		// for identification
		DeltaMtAFs      []*big.Int
		DeltaMtADs      []*big.Int
		DeltaMtADProofs []*zkpaffg.ProofAffg
		ChiMtAFs        []*big.Int
		ChiMtADs        []*big.Int
		ChiMtADProofs   []*zkpaffg.ProofAffg
		R5msgH          []*big.Int
		R5msgProofMul   []*zkpmul.ProofMul
		R5msgProofDec   []*zkpdec.ProofDec
		R5msgDjis       [][]*big.Int
		R5msgFjis       [][]*big.Int
	}

	LocalDump struct {
		Temp     *localTempData
		RoundNum int
		Index    int
	}

	Transcript struct { // for signing identification
		K              *big.Int
		R1msgK         []*big.Int
		ChiShareAlphas []*big.Int
		ChiShareBetas  []*big.Int
		R2msgChiD      []*big.Int

		ChiMtAFs      []*big.Int
		ChiMtADs      []*big.Int
		ChiMtADProofs []*zkpaffg.ProofAffg
	}
)

func NewLocalParty(
	params *tss.Parameters,
	key keygen.LocalPartySaveData,
	out chan<- tss.Message,
	end chan<- *PreSignatureData,
	dump chan<- *LocalDumpPB,
) tss.Party {
	partyCount := len(params.Parties().IDs())
	p := &LocalParty{
		BaseParty: new(tss.BaseParty),
		params:    params,
		keys:      keygen.BuildLocalSaveDataSubset(key, params.Parties().IDs()),
		temp:      localTempData{},
		out:       out,
		end:       end,
		dump:      dump,
	}
	p.startRndNum = 1
	// temp data init
	p.temp.BigWs = make([]*crypto.ECPoint, partyCount)
	p.temp.DeltaShareBetas = make([]*big.Int, partyCount)
	p.temp.ChiShareBetas = make([]*big.Int, partyCount)
	p.temp.DeltaShareAlphas = make([]*big.Int, partyCount)
	p.temp.ChiShareAlphas = make([]*big.Int, partyCount)
	// temp message data init
	p.temp.R1msgG = make([]*big.Int, partyCount)
	p.temp.R1msgK = make([]*big.Int, partyCount)
	p.temp.R1msgProof = make([]*zkpenc.ProofEnc, partyCount)
	p.temp.R2msgBigGammaShare = make([]*crypto.ECPoint, partyCount)
	p.temp.R2msgDeltaD = make([]*big.Int, partyCount)
	p.temp.R2msgDeltaF = make([]*big.Int, partyCount)
	p.temp.R2msgDeltaProof = make([]*zkpaffg.ProofAffg, partyCount)
	p.temp.R2msgChiD = make([]*big.Int, partyCount)
	p.temp.R2msgChiF = make([]*big.Int, partyCount)
	p.temp.R2msgChiProof = make([]*zkpaffg.ProofAffg, partyCount)
	p.temp.R2msgProofLogstar = make([]*zkplogstar.ProofLogstar, partyCount)
	p.temp.R3msgDeltaShare = make([]*big.Int, partyCount)
	p.temp.R3msgBigDeltaShare = make([]*crypto.ECPoint, partyCount)
	p.temp.R3msgProofLogstar = make([]*zkplogstar.ProofLogstar, partyCount)
	// for identification
	p.temp.DeltaMtAFs = make([]*big.Int, partyCount)
	p.temp.DeltaMtADs = make([]*big.Int, partyCount)
	p.temp.DeltaMtADProofs = make([]*zkpaffg.ProofAffg, partyCount)
	p.temp.ChiMtAFs = make([]*big.Int, partyCount)
	p.temp.ChiMtADs = make([]*big.Int, partyCount)
	p.temp.ChiMtADProofs = make([]*zkpaffg.ProofAffg, partyCount)
	p.temp.R5msgH = make([]*big.Int, partyCount)
	p.temp.R5msgProofMul = make([]*zkpmul.ProofMul, partyCount)
	p.temp.R5msgProofDec = make([]*zkpdec.ProofDec, partyCount)
	p.temp.R5msgDjis = make([][]*big.Int, partyCount)
	p.temp.R5msgFjis = make([][]*big.Int, partyCount)

	return p
}

func RestoreLocalParty(
	params *tss.Parameters,
	key keygen.LocalPartySaveData,
	du *LocalDumpPB,
	out chan<- tss.Message,
	end chan<- *PreSignatureData,
	dump chan<- *LocalDumpPB,
) (tss.Party, *tss.Error) {
	p := &LocalParty{
		BaseParty: new(tss.BaseParty),
		params:    params,
		keys:      keygen.BuildLocalSaveDataSubset(key, params.Parties().IDs()),
		temp:      localTempData{},
		out:       out,
		end:       end,
		dump:      dump,
	}
	p.startRndNum = du.UnmarshalRoundNum()
	dtemp, err := du.UnmarshalLocalTemp(p.params.EC())
	if err != nil {
		return nil, tss.NewError(err, TaskName, p.startRndNum, p.PartyID())
	}
	p.temp = *dtemp

	errb := tss.BaseRestore(p, TaskName)
	if errb != nil {
		return nil, errb
	}
	return p, nil
}

func (p *LocalParty) FirstRound() tss.Round {
	newRound := []interface{}{newRound1, newRound2, newRound3, newRound4, newRound5, newRound6}
	return newRound[p.startRndNum-1].(func(*tss.Parameters, *keygen.LocalPartySaveData, *localTempData, chan<- tss.Message, chan<- *PreSignatureData, chan<- *LocalDumpPB) tss.Round)(p.params, &p.keys, &p.temp, p.out, p.end, p.dump)
}

func (p *LocalParty) Start() *tss.Error {
	if p.startRndNum == 1 {
		return tss.BaseStart(p, TaskName, func(round tss.Round) *tss.Error {
			round1, ok := round.(*presign1)
			if !ok {
				return round.WrapError(errors.New("unable to Start(). party is in an unexpected round"))
			}
			if err := round1.prepare(); err != nil {
				return round.WrapError(err)
			}
			return nil
		})
	}
	return tss.BaseStart(p, TaskName)
}

func (p *LocalParty) Update(msg tss.ParsedMessage) (ok bool, err *tss.Error) {
	return tss.BaseUpdate(p, msg, TaskName)
}

func (p *LocalParty) UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool) (bool, *tss.Error) {
	msg, err := tss.ParseWireMessage(wireBytes, from, isBroadcast)
	if err != nil {
		return false, p.WrapError(err)
	}
	return p.Update(msg)
}

func (p *LocalParty) ValidateMessage(msg tss.ParsedMessage) (bool, *tss.Error) {
	if ok, err := p.BaseParty.ValidateMessage(msg); !ok || err != nil {
		return ok, err
	}
	// check that the message's "from index" will fit into the array
	if maxFromIdx := len(p.params.Parties().IDs()) - 1; maxFromIdx < msg.GetFrom().Index {
		return false, p.WrapError(fmt.Errorf("received msg with a sender index too great (%d <= %d)",
			maxFromIdx, msg.GetFrom().Index), msg.GetFrom())
	}
	return true, nil
}

func (p *LocalParty) StoreMessage(msg tss.ParsedMessage) (bool, *tss.Error) {
	// ValidateBasic is cheap; double-check the message here in case the public StoreMessage was called externally
	if ok, err := p.ValidateMessage(msg); !ok || err != nil {
		return ok, err
	}
	fromPIdx := msg.GetFrom().Index

	// switch/case is necessary to store any messages beyond current round
	// this does not handle message replays. we expect the caller to apply replay and spoofing protection.
	switch msg.Content().(type) {
	case *PreSignRound1Message:
		r1msg := msg.Content().(*PreSignRound1Message)
		p.temp.R1msgG[fromPIdx] = r1msg.UnmarshalG()
		p.temp.R1msgK[fromPIdx] = r1msg.UnmarshalK()
		Proof, err := r1msg.UnmarshalEncProof()
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R1msgProof[fromPIdx] = Proof
	case *PreSignRound2Message:
		r2msg := msg.Content().(*PreSignRound2Message)
		BigGammaShare, err := r2msg.UnmarshalBigGammaShare(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R2msgBigGammaShare[fromPIdx] = BigGammaShare
		p.temp.R2msgDeltaD[fromPIdx] = r2msg.UnmarshalDjiDelta()
		p.temp.R2msgDeltaF[fromPIdx] = r2msg.UnmarshalFjiDelta()
		proofDelta, err := r2msg.UnmarshalAffgProofDelta(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R2msgDeltaProof[fromPIdx] = proofDelta
		p.temp.R2msgChiD[fromPIdx] = r2msg.UnmarshalDjiChi()
		p.temp.R2msgChiF[fromPIdx] = r2msg.UnmarshalFjiChi()
		proofChi, err := r2msg.UnmarshalAffgProofChi(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R2msgChiProof[fromPIdx] = proofChi
		proofLogStar, err := r2msg.UnmarshalLogstarProof(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R2msgProofLogstar[fromPIdx] = proofLogStar
	case *PreSignRound3Message:
		r3msg := msg.Content().(*PreSignRound3Message)
		p.temp.R3msgDeltaShare[fromPIdx] = r3msg.UnmarshalDeltaShare()
		BigDeltaShare, err := r3msg.UnmarshalBigDeltaShare(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R3msgBigDeltaShare[fromPIdx] = BigDeltaShare
		proofLogStar, err := r3msg.UnmarshalProofLogstar(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R3msgProofLogstar[fromPIdx] = proofLogStar
	case *IdentificationRound1Message:
		r5msg := msg.Content().(*IdentificationRound1Message)
		p.temp.R5msgH[fromPIdx] = r5msg.UnmarshalH()
		proofMul, err := r5msg.UnmarshalProofMul()
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R5msgProofMul[fromPIdx] = proofMul
		p.temp.R5msgDjis[fromPIdx] = r5msg.UnmarshalDjis()
		p.temp.R5msgFjis[fromPIdx] = r5msg.UnmarshalFjis()
		proofDec, err := r5msg.UnmarshalProofDec()
		if err != nil {
			return false, p.WrapError(err, msg.GetFrom())
		}
		p.temp.R5msgProofDec[fromPIdx] = proofDec
	default: // unrecognised message, just ignore!
		common.Logger.Warningf("unrecognised message ignored: %v", msg)
		return false, nil
	}
	return true, nil
}

func (p *LocalParty) PartyID() *tss.PartyID {
	return p.params.PartyID()
}

func (p *LocalParty) String() string {
	return fmt.Sprintf("id: %s, %s", p.PartyID(), p.BaseParty.String())
}
