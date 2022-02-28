// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package presigning

import (
	"crypto/elliptic"
	"math/big"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	zkpaffg "github.com/binance-chain/tss-lib/crypto/zkp/affg"
	zkpdec "github.com/binance-chain/tss-lib/crypto/zkp/dec"
	zkpenc "github.com/binance-chain/tss-lib/crypto/zkp/enc"
	zkplogstar "github.com/binance-chain/tss-lib/crypto/zkp/logstar"
	zkpmul "github.com/binance-chain/tss-lib/crypto/zkp/mul"
	"github.com/binance-chain/tss-lib/tss"
)

// These messages were generated from Protocol Buffers definitions into ecdsa-signing.pb.go
// The following messages are registered on the Protocol Buffers "wire"

var (
	// Ensure that signing messages implement ValidateBasic
	_ = []tss.MessageContent{
		(*PreSignRound1Message)(nil),
		(*PreSignRound2Message)(nil),
		(*PreSignRound3Message)(nil),
		(*IdentificationRound1Message)(nil),
	}
)

// ----- //
func NewPreSignData(
	index int,
	ssid []byte,
	bigR *crypto.ECPoint,
	kShare *big.Int,
	chiShare *big.Int,
	trans *Transcript,
) *PreSignatureData {
	bigRBzs := bigR.Bytes()

	var KBzs []byte
	if trans.K != nil {
		KBzs = trans.K.Bytes()
	}
	r1msgKBzs := make([][]byte, len(trans.R1msgK))
	for i, item := range trans.R1msgK {
		if item != nil {
			r1msgKBzs[i] = item.Bytes()
		}
	}
	ChiShareAlphasBzs := make([][]byte, len(trans.ChiShareAlphas))
	for i, item := range trans.ChiShareAlphas {
		if item != nil {
			ChiShareAlphasBzs[i] = item.Bytes()
		}
	}
	ChiShareBetasBzs := make([][]byte, len(trans.ChiShareBetas))
	for i, item := range trans.ChiShareBetas {
		if item != nil {
			ChiShareBetasBzs[i] = item.Bytes()
		}
	}
	r2msgChiDBzs := make([][]byte, len(trans.R2msgChiD))
	for i, item := range trans.R2msgChiD {
		if item != nil {
			r2msgChiDBzs[i] = item.Bytes()
		}
	}

	ChiMtAFsBzs := make([][]byte, len(trans.ChiMtAFs))
	for i, item := range trans.ChiMtAFs {
		if item != nil {
			ChiMtAFsBzs[i] = item.Bytes()
		}
	}
	ChiMtADsBzs := make([][]byte, len(trans.ChiMtADs))
	for i, item := range trans.ChiMtADs {
		if item != nil {
			ChiMtADsBzs[i] = item.Bytes()
		}
	}
	ChiMtaDProofsBzs := make([][]byte, len(trans.ChiMtADProofs)*zkpaffg.ProofAffgBytesParts)
	for i, item := range trans.ChiMtADProofs {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				ChiMtaDProofsBzs[i*zkpaffg.ProofAffgBytesParts+j] = itemBzs[j]
			}
		}
	}
	content := &PreSignatureData{
		Index:    int32(index),
		Ssid:     ssid,
		BigR:     bigRBzs[:],
		KShare:   kShare.Bytes(),
		ChiShare: chiShare.Bytes(),

		LRK:              KBzs,
		LRr1MsgK:         r1msgKBzs,
		LRChiShareAlphas: ChiShareAlphasBzs,
		LRChiShareBetas:  ChiShareBetasBzs,
		LRr2MsgChiD:      r2msgChiDBzs,

		LRChiMtAFs:      ChiMtAFsBzs,
		LRChiMtADs:      ChiMtADsBzs,
		LRChiMtADProofs: ChiMtaDProofsBzs,
	}
	return content
}

func (m *PreSignatureData) UnmarshalIndex() int {
	return int(m.GetIndex())
}

func (m *PreSignatureData) UnmarshalSsid() []byte {
	return m.GetSsid()
}

func (m *PreSignatureData) UnmarshalBigR(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetBigR())
}

func (m *PreSignatureData) UnmarshalKShare() *big.Int {
	return new(big.Int).SetBytes(m.GetKShare())
}

func (m *PreSignatureData) UnmarshalChiShare() *big.Int {
	return new(big.Int).SetBytes(m.GetChiShare())
}

func (m *PreSignatureData) UnmarshalTrans(ec elliptic.Curve) (*Transcript, error) {
	KBzs := m.GetLRK()
	var K *big.Int
	if KBzs != nil {
		K = new(big.Int).SetBytes(KBzs)
	}
	r1msgKBzs := m.GetLRr1MsgK()
	r1msgK := make([]*big.Int, len(r1msgKBzs))
	for i := range r1msgK {
		Bzs := r1msgKBzs[i]
		if Bzs != nil {
			r1msgK[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiShareAlphasBzs := m.GetLRChiShareAlphas()
	ChiShareAlphas := make([]*big.Int, len(ChiShareAlphasBzs))
	for i := range ChiShareAlphas {
		Bzs := ChiShareAlphasBzs[i]
		if Bzs != nil {
			ChiShareAlphas[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiShareBetasBzs := m.GetLRChiShareBetas()
	ChiShareBetas := make([]*big.Int, len(ChiShareBetasBzs))
	for i := range ChiShareBetas {
		Bzs := ChiShareBetasBzs[i]
		if Bzs != nil {
			ChiShareBetas[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	r2msgChiDBzs := m.GetLRr2MsgChiD()
	r2msgChiD := make([]*big.Int, len(r2msgChiDBzs))
	for i := range r2msgChiD {
		Bzs := r2msgChiDBzs[i]
		if Bzs != nil {
			r2msgChiD[i] = new(big.Int).SetBytes(Bzs)
		}
	}

	ChiMtAFsBzs := m.GetLRChiMtAFs()
	ChiMtAFs := make([]*big.Int, len(ChiMtAFsBzs))
	for i := range ChiMtAFs {
		Bzs := ChiMtAFsBzs[i]
		if Bzs != nil {
			ChiMtAFs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiMtADsBzs := m.GetLRChiMtADs()
	ChiMtADs := make([]*big.Int, len(ChiMtADsBzs))
	for i := range ChiMtADs {
		Bzs := ChiMtADsBzs[i]
		if Bzs != nil {
			ChiMtADs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiMtADProofsBzs := m.GetLRChiMtADProofs()
	ChiMtADProofs := make([]*zkpaffg.ProofAffg, len(ChiMtADProofsBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range ChiMtADProofs {
		if ChiMtADProofsBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
			item, err := zkpaffg.NewProofFromBytes(ec, ChiMtADProofsBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err != nil {
				return nil, err
			}
			ChiMtADProofs[i] = item
		}
	}

	trans := &Transcript{
		K:              K,
		R1msgK:         r1msgK,
		ChiShareAlphas: ChiShareAlphas,
		ChiShareBetas:  ChiShareBetas,
		R2msgChiD:      r2msgChiD,

		ChiMtAFs:      ChiMtAFs,
		ChiMtADs:      ChiMtADs,
		ChiMtADProofs: ChiMtADProofs,
	}
	return trans, nil
}

func NewPreSignRound1Message(
	to, from *tss.PartyID,
	K *big.Int,
	G *big.Int,
	EncProof *zkpenc.ProofEnc,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		To:          []*tss.PartyID{to},
		IsBroadcast: false,
	}
	pfBz := EncProof.Bytes()
	content := &PreSignRound1Message{
		K:        K.Bytes(),
		G:        G.Bytes(),
		EncProof: pfBz[:],
	}
	msg := tss.NewMessageWrapper(meta, content)
	return tss.NewMessage(meta, content, msg)
}

func (m *PreSignRound1Message) ValidateBasic() bool {
	return m != nil &&
		common.NonEmptyBytes(m.K) &&
		common.NonEmptyBytes(m.G) &&
		common.NonEmptyMultiBytes(m.EncProof, zkpenc.ProofEncBytesParts)
}

func (m *PreSignRound1Message) UnmarshalK() *big.Int {
	return new(big.Int).SetBytes(m.GetK())
}

func (m *PreSignRound1Message) UnmarshalG() *big.Int {
	return new(big.Int).SetBytes(m.GetG())
}

func (m *PreSignRound1Message) UnmarshalEncProof() (*zkpenc.ProofEnc, error) {
	return zkpenc.NewProofFromBytes(m.GetEncProof())
}

// ----- //

func NewPreSignRound2Message(
	to, from *tss.PartyID,
	BigGammaShare *crypto.ECPoint,
	DjiDelta *big.Int,
	FjiDelta *big.Int,
	DjiChi *big.Int,
	FjiChi *big.Int,
	AffgProofDelta *zkpaffg.ProofAffg,
	AffgProofChi *zkpaffg.ProofAffg,
	LogstarProof *zkplogstar.ProofLogstar,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		To:          []*tss.PartyID{to},
		IsBroadcast: false,
	}
	BigGammaBytes := BigGammaShare.Bytes()
	AffgDeltaBz := AffgProofDelta.Bytes()
	AffgChiBz := AffgProofChi.Bytes()
	LogstarBz := LogstarProof.Bytes()
	content := &PreSignRound2Message{
		BigGammaShare:  BigGammaBytes[:],
		DjiDelta:       DjiDelta.Bytes(),
		FjiDelta:       FjiDelta.Bytes(),
		DjiChi:         DjiChi.Bytes(),
		FjiChi:         FjiChi.Bytes(),
		AffgProofDelta: AffgDeltaBz[:],
		AffgProofChi:   AffgChiBz[:],
		LogstarProof:   LogstarBz[:],
	}
	msg := tss.NewMessageWrapper(meta, content)
	return tss.NewMessage(meta, content, msg)
}

func (m *PreSignRound2Message) ValidateBasic() bool {
	return m != nil &&
		common.NonEmptyMultiBytes(m.BigGammaShare, 2) &&
		common.NonEmptyBytes(m.DjiDelta) &&
		common.NonEmptyBytes(m.FjiDelta) &&
		common.NonEmptyBytes(m.DjiChi) &&
		common.NonEmptyBytes(m.FjiChi) &&
		common.NonEmptyMultiBytes(m.AffgProofDelta, zkpaffg.ProofAffgBytesParts) &&
		common.NonEmptyMultiBytes(m.AffgProofChi, zkpaffg.ProofAffgBytesParts) &&
		common.NonEmptyMultiBytes(m.LogstarProof, zkplogstar.ProofLogstarBytesParts)
}

func (m *PreSignRound2Message) UnmarshalBigGammaShare(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetBigGammaShare())
}

func (m *PreSignRound2Message) UnmarshalDjiDelta() *big.Int {
	return new(big.Int).SetBytes(m.GetDjiDelta())
}

func (m *PreSignRound2Message) UnmarshalFjiDelta() *big.Int {
	return new(big.Int).SetBytes(m.GetFjiDelta())
}

func (m *PreSignRound2Message) UnmarshalDjiChi() *big.Int {
	return new(big.Int).SetBytes(m.GetDjiChi())
}

func (m *PreSignRound2Message) UnmarshalFjiChi() *big.Int {
	return new(big.Int).SetBytes(m.GetFjiChi())
}

func (m *PreSignRound2Message) UnmarshalAffgProofDelta(ec elliptic.Curve) (*zkpaffg.ProofAffg, error) {
	return zkpaffg.NewProofFromBytes(ec, m.GetAffgProofDelta())
}

func (m *PreSignRound2Message) UnmarshalAffgProofChi(ec elliptic.Curve) (*zkpaffg.ProofAffg, error) {
	return zkpaffg.NewProofFromBytes(ec, m.GetAffgProofChi())
}

func (m *PreSignRound2Message) UnmarshalLogstarProof(ec elliptic.Curve) (*zkplogstar.ProofLogstar, error) {
	return zkplogstar.NewProofFromBytes(ec, m.GetLogstarProof())
}

// ----- //

func NewPreSignRound3Message(
	to, from *tss.PartyID,
	DeltaShare *big.Int,
	BigDeltaShare *crypto.ECPoint,
	ProofLogstar *zkplogstar.ProofLogstar,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		To:          []*tss.PartyID{to},
		IsBroadcast: false,
	}
	BigDeltaShareBzs := BigDeltaShare.Bytes()
	ProofBz := ProofLogstar.Bytes()
	content := &PreSignRound3Message{
		DeltaShare:    DeltaShare.Bytes(),
		BigDeltaShare: BigDeltaShareBzs[:],
		ProofLogstar:  ProofBz[:],
	}
	msg := tss.NewMessageWrapper(meta, content)
	return tss.NewMessage(meta, content, msg)
}

func (m *PreSignRound3Message) ValidateBasic() bool {
	return m != nil &&
		common.NonEmptyBytes(m.DeltaShare) &&
		common.NonEmptyMultiBytes(m.BigDeltaShare, 2) &&
		common.NonEmptyMultiBytes(m.ProofLogstar, zkplogstar.ProofLogstarBytesParts)
}

func (m *PreSignRound3Message) UnmarshalDeltaShare() *big.Int {
	return new(big.Int).SetBytes(m.GetDeltaShare())
}

func (m *PreSignRound3Message) UnmarshalBigDeltaShare(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetBigDeltaShare())
}

func (m *PreSignRound3Message) UnmarshalProofLogstar(ec elliptic.Curve) (*zkplogstar.ProofLogstar, error) {
	return zkplogstar.NewProofFromBytes(ec, m.GetProofLogstar())
}

// ----- //

func NewIdentificationRound1Message(
	to, from *tss.PartyID,
	H *big.Int,
	MulProof *zkpmul.ProofMul,
	Djis []*big.Int,
	Fjis []*big.Int,
	DjiProofs []*zkpaffg.ProofAffg,
	DecProof *zkpdec.ProofDec,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		To:          []*tss.PartyID{to},
		IsBroadcast: false,
	}
	MulProofBzs := MulProof.Bytes()
	DjisBzs := make([][]byte, len(Djis))
	for i, item := range Djis {
		if item != nil {
			DjisBzs[i] = Djis[i].Bytes()
		}
	}
	FjisBzs := make([][]byte, len(Fjis))
	for i, item := range Fjis {
		if item != nil {
			FjisBzs[i] = Fjis[i].Bytes()
		}
	}
	DjiProofsBzs := make([][]byte, len(DjiProofs)*zkpaffg.ProofAffgBytesParts)
	DecProofBzs := DecProof.Bytes()
	for i, item := range DjiProofs {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				DjiProofsBzs[i*zkpenc.ProofEncBytesParts+j] = itemBzs[j]
			}
		}
	}
	content := &IdentificationRound1Message{
		H:         H.Bytes(),
		MulProof:  MulProofBzs[:],
		Djis:      DjisBzs,
		Fjis:      FjisBzs,
		DjiProofs: DjiProofsBzs,
		DecProof:  DecProofBzs[:],
	}
	msg := tss.NewMessageWrapper(meta, content)
	return tss.NewMessage(meta, content, msg)
}

func (m *IdentificationRound1Message) ValidateBasic() bool {
	return m != nil &&
		common.NonEmptyBytes(m.H) &&
		common.NonEmptyMultiBytes(m.MulProof, zkpmul.ProofMulBytesParts) &&
		// TODO not empty excluding own index
		//common.NonEmptyMultiBytes(m.Djis) &&
		//common.NonEmptyMultiBytes(m.Fjis) &&
		//common.NonEmptyMultiBytes(m.DjiProofs, zkpaffg.ProofAffgBytesParts) &&
		common.NonEmptyMultiBytes(m.DecProof, zkpdec.ProofDecBytesParts)
}

func (m *IdentificationRound1Message) UnmarshalH() *big.Int {
	return new(big.Int).SetBytes(m.GetH())
}

func (m *IdentificationRound1Message) UnmarshalProofMul() (*zkpmul.ProofMul, error) {
	return zkpmul.NewProofFromBytes(m.GetMulProof())
}

func (m *IdentificationRound1Message) UnmarshalDjis() []*big.Int {
	DjisBzs := m.GetDjis()
	Djis := make([]*big.Int, len(DjisBzs))
	for i := range Djis {
		Bzs := DjisBzs[i]
		if Bzs != nil {
			Djis[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	return Djis
}

func (m *IdentificationRound1Message) UnmarshalFjis() []*big.Int {
	FjisBzs := m.GetFjis()
	Fjis := make([]*big.Int, len(FjisBzs))
	for i := range Fjis {
		Bzs := FjisBzs[i]
		if Bzs != nil {
			Fjis[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	return Fjis
}

func (m *IdentificationRound1Message) UnmarshalDjiProofs(ec elliptic.Curve) []*zkpaffg.ProofAffg {
	DjiProofsBzs := m.GetDjiProofs()
	DjiProofs := make([]*zkpaffg.ProofAffg, len(DjiProofsBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range DjiProofs {
		if DjiProofsBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
			item, err := zkpaffg.NewProofFromBytes(ec, DjiProofsBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err == nil { // continue if error occurs
				DjiProofs[i] = item
			}
		}
	}
	return DjiProofs
}

func (m *IdentificationRound1Message) UnmarshalProofDec() (*zkpdec.ProofDec, error) {
	return zkpdec.NewProofFromBytes(m.GetDecProof())
}

func NewLocalDumpPB(
	Index int,
	RoundNum int,
	LocalTemp *localTempData,
) *LocalDumpPB {
	var WBzs []byte
	if LocalTemp.W != nil {
		WBzs = LocalTemp.W.Bytes()
	}
	BigWsBzs := make([][]byte, len(LocalTemp.BigWs)*2)
	for i, item := range LocalTemp.BigWs {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < 2; j++ {
				BigWsBzs[i*2+j] = itemBzs[j]
			}
		}
	}
	var KShareBzs []byte
	if LocalTemp.KShare != nil {
		KShareBzs = LocalTemp.KShare.Bytes()
	}

	var BigGammaShareBzs [][]byte
	if LocalTemp.BigGammaShare != nil {
		Bzs := LocalTemp.BigGammaShare.Bytes()
		BigGammaShareBzs = Bzs[:]
	}
	var KBzs []byte
	if LocalTemp.K != nil {
		KBzs = LocalTemp.K.Bytes()
	}
	var GBzs []byte
	if LocalTemp.G != nil {
		GBzs = LocalTemp.G.Bytes()
	}
	var KNonceBzs []byte
	if LocalTemp.KNonce != nil {
		KNonceBzs = LocalTemp.KNonce.Bytes()
	}
	var GNonceBzs []byte
	if LocalTemp.GNonce != nil {
		GNonceBzs = LocalTemp.GNonce.Bytes()
	}

	var GammaShareBzs []byte
	if LocalTemp.GammaShare != nil {
		GammaShareBzs = LocalTemp.GammaShare.Bytes()
	}
	DeltaShareBetasBzs := make([][]byte, len(LocalTemp.DeltaShareBetas))
	for i, item := range LocalTemp.DeltaShareBetas {
		if item != nil {
			DeltaShareBetasBzs[i] = item.Bytes()
		}
	}
	ChiShareBetasBzs := make([][]byte, len(LocalTemp.ChiShareBetas))
	for i, item := range LocalTemp.ChiShareBetas {
		if item != nil {
			ChiShareBetasBzs[i] = item.Bytes()
		}
	}

	var BigGammaBzs [][]byte
	if LocalTemp.BigGamma != nil {
		Bzs := LocalTemp.BigGamma.Bytes()
		BigGammaBzs = Bzs[:]
	}
	DeltaShareAlphasBzs := make([][]byte, len(LocalTemp.DeltaShareAlphas))
	for i, item := range LocalTemp.DeltaShareAlphas {
		if item != nil {
			DeltaShareAlphasBzs[i] = item.Bytes()
		}
	}
	ChiShareAlphasBzs := make([][]byte, len(LocalTemp.ChiShareAlphas))
	for i, item := range LocalTemp.ChiShareAlphas {
		if item != nil {
			ChiShareAlphasBzs[i] = item.Bytes()
		}
	}
	var DeltaShareBzs []byte
	if LocalTemp.DeltaShare != nil {
		DeltaShareBzs = LocalTemp.DeltaShare.Bytes()
	}
	var ChiShareBzs []byte
	if LocalTemp.ChiShare != nil {
		ChiShareBzs = LocalTemp.ChiShare.Bytes()
	}
	var BigDeltaShareBzs [][]byte
	if LocalTemp.BigDeltaShare != nil {
		Bzs := LocalTemp.BigDeltaShare.Bytes()
		BigDeltaShareBzs = Bzs[:]
	}

	var BigRBzs [][]byte
	if LocalTemp.BigR != nil {
		Bzs := LocalTemp.BigR.Bytes()
		BigRBzs = Bzs[:]
	}
	var RxBzs []byte
	if LocalTemp.Rx != nil {
		RxBzs = LocalTemp.Rx.Bytes()
	}
	var SigmaShareBzs []byte
	if LocalTemp.SigmaShare != nil {
		SigmaShareBzs = LocalTemp.SigmaShare.Bytes()
	}

	R1msgGBzs := make([][]byte, len(LocalTemp.R1msgG))
	for i, item := range LocalTemp.R1msgG {
		if item != nil {
			R1msgGBzs[i] = item.Bytes()
		}
	}
	R1msgKBzs := make([][]byte, len(LocalTemp.R1msgK))
	for i, item := range LocalTemp.R1msgK {
		if item != nil {
			R1msgKBzs[i] = item.Bytes()
		}
	}
	R1msgProofBzs := make([][]byte, len(LocalTemp.R1msgProof)*zkpenc.ProofEncBytesParts)
	for i, item := range LocalTemp.R1msgProof {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < zkpenc.ProofEncBytesParts; j++ {
				R1msgProofBzs[i*zkpenc.ProofEncBytesParts+j] = itemBzs[j]
			}
		}
	}

	R2msgBigGammaShareBzs := make([][]byte, len(LocalTemp.R2msgBigGammaShare)*2)
	for i, item := range LocalTemp.R2msgBigGammaShare {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < 2; j++ {
				R2msgBigGammaShareBzs[i*2+j] = itemBzs[j]
			}
		}
	}
	R2msgDeltaDBzs := make([][]byte, len(LocalTemp.R2msgDeltaD))
	for i, item := range LocalTemp.R2msgDeltaD {
		if item != nil {
			R2msgDeltaDBzs[i] = item.Bytes()
		}
	}
	R2msgDeltaFBzs := make([][]byte, len(LocalTemp.R2msgDeltaF))
	for i, item := range LocalTemp.R2msgDeltaF {
		if item != nil {
			R2msgDeltaFBzs[i] = item.Bytes()
		}
	}
	R2msgDeltaProofBzs := make([][]byte, len(LocalTemp.R2msgDeltaProof)*zkpaffg.ProofAffgBytesParts)
	for i, item := range LocalTemp.R2msgDeltaProof {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				R2msgDeltaProofBzs[i*zkpaffg.ProofAffgBytesParts+j] = itemBzs[j]
			}
		}
	}
	R2msgChiDBzs := make([][]byte, len(LocalTemp.R2msgChiD))
	for i, item := range LocalTemp.R2msgChiD {
		if item != nil {
			R2msgChiDBzs[i] = item.Bytes()
		}
	}
	R2msgChiFBzs := make([][]byte, len(LocalTemp.R2msgChiF))
	for i, item := range LocalTemp.R2msgChiF {
		if item != nil {
			R2msgChiFBzs[i] = item.Bytes()
		}
	}
	R2msgChiProofBzs := make([][]byte, len(LocalTemp.R2msgChiProof)*zkpaffg.ProofAffgBytesParts)
	for i, item := range LocalTemp.R2msgChiProof {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				R2msgChiProofBzs[i*zkpaffg.ProofAffgBytesParts+j] = itemBzs[j]
			}
		}
	}
	R2msgProofLogstarBzs := make([][]byte, len(LocalTemp.R2msgProofLogstar)*zkplogstar.ProofLogstarBytesParts)
	for i, item := range LocalTemp.R2msgProofLogstar {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkplogstar.ProofLogstarBytesParts; j++ {
				R2msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts+j] = itemBzs[j]
			}
		}
	}

	R3msgDeltaShareBzs := make([][]byte, len(LocalTemp.R3msgDeltaShare))
	for i, item := range LocalTemp.R3msgDeltaShare {
		if item != nil {
			R3msgDeltaShareBzs[i] = item.Bytes()
		}
	}
	R3msgBigDeltaShareBzs := make([][]byte, len(LocalTemp.R3msgBigDeltaShare)*2)
	for i, item := range LocalTemp.R3msgBigDeltaShare {
		if item != nil {
			itemBzs := item.Bytes()
			for j := 0; j < 2; j++ {
				R3msgBigDeltaShareBzs[i*2+j] = itemBzs[j]
			}
		}
	}
	R3msgProofLogstarBzs := make([][]byte, len(LocalTemp.R3msgProofLogstar)*zkplogstar.ProofLogstarBytesParts)
	for i, item := range LocalTemp.R3msgProofLogstar {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkplogstar.ProofLogstarBytesParts; j++ {
				R3msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts+j] = itemBzs[j]
			}
		}
	}

	DeltaMtAFsBzs := make([][]byte, len(LocalTemp.DeltaMtAFs))
	for i, item := range LocalTemp.DeltaMtAFs {
		if item != nil {
			DeltaMtAFsBzs[i] = item.Bytes()
		}
	}
	DeltaMtADsBzs := make([][]byte, len(LocalTemp.DeltaMtADs))
	for i, item := range LocalTemp.DeltaMtADs {
		if item != nil {
			DeltaMtADsBzs[i] = item.Bytes()
		}
	}
	DeltaMtaDProofsBzs := make([][]byte, len(LocalTemp.DeltaMtADProofs)*zkpaffg.ProofAffgBytesParts)
	for i, item := range LocalTemp.DeltaMtADProofs {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				DeltaMtaDProofsBzs[i*zkpaffg.ProofAffgBytesParts+j] = itemBzs[j]
			}
		}
	}
	ChiMtAFsBzs := make([][]byte, len(LocalTemp.ChiMtAFs))
	for i, item := range LocalTemp.ChiMtAFs {
		if item != nil {
			ChiMtAFsBzs[i] = item.Bytes()
		}
	}
	ChiMtADsBzs := make([][]byte, len(LocalTemp.ChiMtADs))
	for i, item := range LocalTemp.ChiMtADs {
		if item != nil {
			ChiMtADsBzs[i] = item.Bytes()
		}
	}
	ChiMtaDProofsBzs := make([][]byte, len(LocalTemp.ChiMtADProofs)*zkpaffg.ProofAffgBytesParts)
	for i, item := range LocalTemp.ChiMtADProofs {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpaffg.ProofAffgBytesParts; j++ {
				ChiMtaDProofsBzs[i*zkpaffg.ProofAffgBytesParts+j] = itemBzs[j]
			}
		}
	}
	R5msgHBzs := make([][]byte, len(LocalTemp.R5msgH))
	for i, item := range LocalTemp.R5msgH {
		if item != nil {
			R5msgHBzs[i] = item.Bytes()
		}
	}
	R5msgProofMulBzs := make([][]byte, len(LocalTemp.R5msgProofMul)*zkpmul.ProofMulBytesParts)
	for i, item := range LocalTemp.R5msgProofMul {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpmul.ProofMulBytesParts; j++ {
				R5msgProofMulBzs[i*zkpmul.ProofMulBytesParts+j] = itemBzs[j]
			}
		}
	}
	//r5msgDeltaShareEncBzs := make([][]byte, len(LocalTemp.r5msgDeltaShareEnc))
	//for i, item := range LocalTemp.r5msgDeltaShareEnc {
	//	if item != nil {
	//		r5msgDeltaShareEncBzs[i] = item.Bytes()
	//	}
	//}
	R5msgProofDecBzs := make([][]byte, len(LocalTemp.R5msgProofDec)*zkpdec.ProofDecBytesParts)
	for i, item := range LocalTemp.R5msgProofDec {
		if item != nil && item.ValidateBasic() {
			itemBzs := item.Bytes()
			for j := 0; j < zkpdec.ProofDecBytesParts; j++ {
				R5msgProofDecBzs[i*zkpdec.ProofDecBytesParts+j] = itemBzs[j]
			}
		}
	}
	R5msgDjiLen := len(LocalTemp.R5msgDjis)
	R5msgDjisBzs := make([][]byte, R5msgDjiLen*R5msgDjiLen)
	for i, row := range LocalTemp.R5msgDjis {
		for j, item := range row {
			if item != nil {
				R5msgDjisBzs[i*R5msgDjiLen+j] = item.Bytes()
			}
		}
	}
	R5msgFjiLen := len(LocalTemp.R5msgFjis)
	R5msgFjisBzs := make([][]byte, R5msgFjiLen*R5msgFjiLen)
	for i, row := range LocalTemp.R5msgFjis {
		for j, item := range row {
			if item != nil {
				R5msgFjisBzs[i*R5msgFjiLen+j] = item.Bytes()
			}
		}
	}
	//r5msgQ3EncBzs := make([][]byte, len(LocalTemp.r5msgQ3Enc))
	//for i, item := range LocalTemp.r5msgQ3Enc {
	//	if item != nil {
	//		r5msgQ3EncBzs[i] = item.Bytes()
	//	}
	//}

	content := &LocalDumpPB{
		Index:    int32(Index),
		RoundNum: int32(RoundNum),

		LTssid:   LocalTemp.Ssid,
		LTw:      WBzs,
		LTBigWs:  BigWsBzs,
		LTKShare: KShareBzs,

		LTBigGammaShare: BigGammaShareBzs,
		LTK:             KBzs,
		LTG:             GBzs,
		LTKNonce:        KNonceBzs,
		LTGNonce:        GNonceBzs,

		LTGammaShare:      GammaShareBzs,
		LTDeltaShareBetas: DeltaShareBetasBzs,
		LTChiShareBetas:   ChiShareBetasBzs,

		LTBigGamma:         BigGammaBzs,
		LTDeltaShareAlphas: DeltaShareAlphasBzs,
		LTChiShareAlphas:   ChiShareAlphasBzs,
		LTDeltaShare:       DeltaShareBzs,
		LTChiShare:         ChiShareBzs,
		LTBigDeltaShare:    BigDeltaShareBzs,

		LTBigR:       BigRBzs,
		LTRx:         RxBzs,
		LTSigmaShare: SigmaShareBzs,

		LTr1MsgG:     R1msgGBzs,
		LTr1MsgK:     R1msgKBzs,
		LTr1MsgProof: R1msgProofBzs,

		LTr2MsgBigGammaShare: R2msgBigGammaShareBzs,
		LTr2MsgDeltaD:        R2msgDeltaDBzs,
		LTr2MsgDeltaF:        R2msgDeltaFBzs,
		LTr2MsgDeltaProof:    R2msgDeltaProofBzs,
		LTr2MsgChiD:          R2msgChiDBzs,
		LTr2MsgChiF:          R2msgChiFBzs,
		LTr2MsgChiProof:      R2msgChiProofBzs,
		LTr2MsgProofLogstar:  R2msgProofLogstarBzs,

		LTr3MsgDeltaShare:    R3msgDeltaShareBzs,
		LTr3MsgBigDeltaShare: R3msgBigDeltaShareBzs,
		LTr3MsgProofLogstar:  R3msgProofLogstarBzs,

		LTDeltaMtAFs:      DeltaMtAFsBzs,
		LTDeltaMtADs:      DeltaMtADsBzs,
		LDDeltaMtADProofs: DeltaMtaDProofsBzs,
		LTChiMtAFs:        ChiMtAFsBzs,
		LTChiMtADs:        ChiMtADsBzs,
		LTChiMtADProofs:   ChiMtaDProofsBzs,
		LTr5MsgH:          R5msgHBzs,
		LTr5MsgProofMul:   R5msgProofMulBzs,
		//LTr6MsgDeltaShareEnc: r6msgDeltaShareEncBzs,
		LTr5MsgProofDec: R5msgProofDecBzs,
		LTr5MsgDjis:     R5msgDjisBzs,
		LTr5MsgFjis:     R5msgFjisBzs,
		//LTr5MsgQ3Enc:         r5msgQ3EncBzs,
	}
	return content
}

func (m *LocalDumpPB) UnmarshalIndex() int {
	return int(m.GetIndex())
}

func (m *LocalDumpPB) UnmarshalRoundNum() int {
	return int(m.GetRoundNum())
}

func (m *LocalDumpPB) UnmarshalLocalTemp(ec elliptic.Curve) (*localTempData, error) {
	Ssid := m.GetLTssid()
	WBzs := m.GetLTw()
	var W *big.Int
	if len(WBzs) > 0 {
		W = new(big.Int).SetBytes(WBzs)
	}
	BigWsBzs := m.GetLTBigWs()
	BigWs := make([]*crypto.ECPoint, len(BigWsBzs)/2)
	for i := range BigWs {
		if BigWsBzs[i*2] != nil {
			item, err := crypto.NewECPointFromBytes(ec, BigWsBzs[(i*2):(i*2+2)])
			if err != nil {
				return nil, err
			}
			BigWs[i] = item
		}
	}
	KShareBzs := m.GetLTKShare()
	var KShare *big.Int
	//if KShareBzs != nil {
	if len(KShareBzs) > 0 {
		KShare = new(big.Int).SetBytes(KShareBzs)
	}

	BigGammaShareBzs := m.GetLTBigGammaShare()
	var BigGammaShare *crypto.ECPoint
	//if BigGammaShareBzs != nil {
	if len(BigGammaShareBzs) > 0 {
		item, err := crypto.NewECPointFromBytes(ec, BigGammaShareBzs)
		if err != nil {
			return nil, err
		}
		BigGammaShare = item
	}
	KBzs := m.GetLTK()
	var K *big.Int
	//if KBzs != nil {
	if len(KBzs) > 0 {
		K = new(big.Int).SetBytes(KBzs)
	}
	GBzs := m.GetLTG()
	var G *big.Int
	//if GBzs != nil {
	if len(GBzs) > 0 {
		G = new(big.Int).SetBytes(GBzs)
	}
	KNonceBzs := m.GetLTKNonce()
	var KNonce *big.Int
	//if KNonceBzs != nil {
	if len(KNonceBzs) > 0 {
		KNonce = new(big.Int).SetBytes(KNonceBzs)
	}
	GNonceBzs := m.GetLTGNonce()
	var GNonce *big.Int
	//if GNonceBzs != nil {
	if len(GNonceBzs) > 0 {
		GNonce = new(big.Int).SetBytes(GNonceBzs)
	}

	GammaShareBzs := m.GetLTGammaShare()
	var GammaShare *big.Int
	//if GammaShareBzs != nil {
	if len(GammaShareBzs) > 0 {
		GammaShare = new(big.Int).SetBytes(GammaShareBzs)
	}
	DeltaShareBetasBzs := m.GetLTDeltaShareBetas()
	DeltaShareBetas := make([]*big.Int, len(DeltaShareBetasBzs))
	for i := range DeltaShareBetas {
		Bzs := DeltaShareBetasBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			DeltaShareBetas[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiShareBetasBzs := m.GetLTChiShareBetas()
	ChiShareBetas := make([]*big.Int, len(ChiShareBetasBzs))
	for i := range ChiShareBetas {
		Bzs := ChiShareBetasBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			ChiShareBetas[i] = new(big.Int).SetBytes(Bzs)
		}
	}

	BigGammaBzs := m.GetLTBigGamma()
	var BigGamma *crypto.ECPoint
	if len(BigGammaBzs) > 0 {
		item, err := crypto.NewECPointFromBytes(ec, BigGammaBzs)
		if err != nil {
			return nil, err
		}
		BigGamma = item
	}
	DeltaShareAlphasBzs := m.GetLTDeltaShareAlphas()
	DeltaShareAlphas := make([]*big.Int, len(DeltaShareAlphasBzs))
	for i := range DeltaShareAlphas {
		Bzs := DeltaShareAlphasBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			DeltaShareAlphas[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiShareAlphasBzs := m.GetLTChiShareAlphas()
	ChiShareAlphas := make([]*big.Int, len(ChiShareAlphasBzs))
	for i := range ChiShareAlphas {
		Bzs := ChiShareAlphasBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			ChiShareAlphas[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	DeltaShareBzs := m.GetLTDeltaShare()
	var DeltaShare *big.Int
	//if DeltaShareBzs != nil {
	if len(DeltaShareBzs) > 0 {
		DeltaShare = new(big.Int).SetBytes(DeltaShareBzs)
	}
	ChiShareBzs := m.GetLTChiShare()
	var ChiShare *big.Int
	//if ChiShareBzs != nil {
	if len(ChiShareBzs) > 0 {
		ChiShare = new(big.Int).SetBytes(ChiShareBzs)
	}
	BigDeltaShareBzs := m.GetLTBigDeltaShare()
	var BigDeltaShare *crypto.ECPoint
	//if BigDeltaShareBzs != nil {
	if len(BigDeltaShareBzs) > 0 {
		item, err := crypto.NewECPointFromBytes(ec, BigDeltaShareBzs)
		if err != nil {
			return nil, err
		}
		BigDeltaShare = item
	}

	BigRBzs := m.GetLTBigR()
	var BigR *crypto.ECPoint
	//if BigRBzs != nil {
	if len(BigRBzs) > 0 {
		item, err := crypto.NewECPointFromBytes(ec, BigRBzs)
		if err != nil {
			return nil, err
		}
		BigR = item
	}
	RxBzs := m.GetLTRx()
	var Rx *big.Int
	//if RxBzs != nil {
	if len(RxBzs) > 0 {
		Rx = new(big.Int).SetBytes(RxBzs)
	}
	SigmaShareBzs := m.GetLTSigmaShare()
	var SigmaShare *big.Int
	//if SigmaShareBzs != nil {
	if len(SigmaShareBzs) > 0 {
		SigmaShare = new(big.Int).SetBytes(SigmaShareBzs)
	}

	R1msgGBzs := m.GetLTr1MsgG()
	R1msgG := make([]*big.Int, len(R1msgGBzs))
	for i := range R1msgG {
		Bzs := R1msgGBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R1msgG[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R1msgKBzs := m.GetLTr1MsgK()
	R1msgK := make([]*big.Int, len(R1msgKBzs))
	for i := range R1msgK {
		Bzs := R1msgKBzs[i]
		if len(Bzs) > 0 {
			R1msgK[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R1msgProofBzs := m.GetLTr1MsgProof()
	R1msgProof := make([]*zkpenc.ProofEnc, len(R1msgProofBzs)/zkpenc.ProofEncBytesParts)
	for i := range R1msgProof {
		//if R1msgProofBzs[i*zkpenc.ProofEncBytesParts] != nil {
		if len(R1msgProofBzs[i*zkpenc.ProofEncBytesParts]) > 0 {
			item, err := zkpenc.NewProofFromBytes(R1msgProofBzs[(i * zkpenc.ProofEncBytesParts):(i*zkpenc.ProofEncBytesParts + zkpenc.ProofEncBytesParts)])
			if err != nil {
				return nil, err
			}
			R1msgProof[i] = item
		}
	}

	R2msgBigGammaShareBzs := m.GetLTr2MsgBigGammaShare()
	R2msgBigGammaShare := make([]*crypto.ECPoint, len(R2msgBigGammaShareBzs)/2)
	for i := range R2msgBigGammaShare {
		//if R2msgBigGammaShareBzs[i*2] != nil {
		if len(R2msgBigGammaShareBzs[i*2]) > 0 {
			item, err := crypto.NewECPointFromBytes(ec, R2msgBigGammaShareBzs[(i*2):(i*2+2)])
			if err != nil {
				return nil, err
			}
			R2msgBigGammaShare[i] = item
		}
	}
	R2msgDeltaDBzs := m.GetLTr2MsgDeltaD()
	R2msgDeltaD := make([]*big.Int, len(R2msgDeltaDBzs))
	for i := range R2msgDeltaD {
		Bzs := R2msgDeltaDBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R2msgDeltaD[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R2msgDeltaFBzs := m.GetLTr2MsgDeltaF()
	R2msgDeltaF := make([]*big.Int, len(R2msgDeltaFBzs))
	for i := range R2msgDeltaF {
		Bzs := R2msgDeltaFBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R2msgDeltaF[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R2msgDeltaProofBzs := m.GetLTr2MsgDeltaProof()
	R2msgDeltaProof := make([]*zkpaffg.ProofAffg, len(R2msgDeltaProofBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range R2msgDeltaProof {
		//if R2msgDeltaProofBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
		if len(R2msgDeltaProofBzs[i*zkpaffg.ProofAffgBytesParts]) > 0 {
			item, err := zkpaffg.NewProofFromBytes(ec, R2msgDeltaProofBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err != nil {
				return nil, err
			}
			R2msgDeltaProof[i] = item
		}
	}
	R2msgChiDBzs := m.GetLTr2MsgChiD()
	R2msgChiD := make([]*big.Int, len(R2msgChiDBzs))
	for i := range R2msgChiD {
		Bzs := R2msgChiDBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R2msgChiD[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R2msgChiFBzs := m.GetLTr2MsgChiF()
	R2msgChiF := make([]*big.Int, len(R2msgChiFBzs))
	for i := range R2msgChiF {
		Bzs := R2msgChiFBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R2msgChiF[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R2msgChiProofBzs := m.GetLTr2MsgChiProof()
	R2msgChiProof := make([]*zkpaffg.ProofAffg, len(R2msgChiProofBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range R2msgDeltaProof {
		//if R2msgChiProofBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
		if len(R2msgChiProofBzs[i*zkpaffg.ProofAffgBytesParts]) > 0 {
			item, err := zkpaffg.NewProofFromBytes(ec, R2msgChiProofBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err != nil {
				return nil, err
			}
			R2msgChiProof[i] = item
		}
	}
	R2msgProofLogstarBzs := m.GetLTr2MsgProofLogstar()
	R2msgProofLogstar := make([]*zkplogstar.ProofLogstar, len(R2msgProofLogstarBzs)/zkplogstar.ProofLogstarBytesParts)
	for i := range R2msgProofLogstar {
		//if R2msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts] != nil {
		if len(R2msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts]) > 0 {
			item, err := zkplogstar.NewProofFromBytes(ec, R2msgProofLogstarBzs[(i*zkplogstar.ProofLogstarBytesParts):(i*zkplogstar.ProofLogstarBytesParts+zkplogstar.ProofLogstarBytesParts)])
			if err != nil {
				return nil, err
			}
			R2msgProofLogstar[i] = item
		}
	}

	R3msgDeltaShareBzs := m.GetLTr3MsgDeltaShare()
	R3msgDeltaShare := make([]*big.Int, len(R3msgDeltaShareBzs))
	for i := range R3msgDeltaShare {
		Bzs := R3msgDeltaShareBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R3msgDeltaShare[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R3msgBigDeltaShareBzs := m.GetLTr3MsgBigDeltaShare()
	R3msgBigDeltaShare := make([]*crypto.ECPoint, len(R3msgBigDeltaShareBzs)/2)
	for i := range R3msgBigDeltaShare {
		//if R3msgBigDeltaShareBzs[i*2] != nil {
		if len(R3msgBigDeltaShareBzs[i*2]) > 0 {
			item, err := crypto.NewECPointFromBytes(ec, R3msgBigDeltaShareBzs[(i*2):(i*2+2)])
			if err != nil {
				return nil, err
			}
			R3msgBigDeltaShare[i] = item
		}
	}
	R3msgProofLogstarBzs := m.GetLTr3MsgProofLogstar()
	R3msgProofLogstar := make([]*zkplogstar.ProofLogstar, len(R3msgProofLogstarBzs)/zkplogstar.ProofLogstarBytesParts)
	for i := range R3msgProofLogstar {
		//if R3msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts] != nil {
		if len(R3msgProofLogstarBzs[i*zkplogstar.ProofLogstarBytesParts]) > 0 {
			item, err := zkplogstar.NewProofFromBytes(ec, R3msgProofLogstarBzs[(i*zkplogstar.ProofLogstarBytesParts):(i*zkplogstar.ProofLogstarBytesParts+zkplogstar.ProofLogstarBytesParts)])
			if err != nil {
				return nil, err
			}
			R3msgProofLogstar[i] = item
		}
	}

	DeltaMtAFsBzs := m.GetLTDeltaMtAFs()
	DeltaMtAFs := make([]*big.Int, len(DeltaMtAFsBzs))
	for i := range DeltaMtAFs {
		Bzs := DeltaMtAFsBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			DeltaMtAFs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	DeltaMtADsBzs := m.GetLTDeltaMtADs()
	DeltaMtADs := make([]*big.Int, len(DeltaMtADsBzs))
	for i := range DeltaMtADs {
		Bzs := DeltaMtADsBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			DeltaMtADs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	DeltaMtADProofsBzs := m.GetLDDeltaMtADProofs()
	DeltaMtADProofs := make([]*zkpaffg.ProofAffg, len(DeltaMtADProofsBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range DeltaMtADProofs {
		//if DeltaMtADProofsBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
		if len(DeltaMtADProofsBzs[i*zkpaffg.ProofAffgBytesParts]) > 0 {
			item, err := zkpaffg.NewProofFromBytes(ec, DeltaMtADProofsBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err != nil {
				return nil, err
			}
			DeltaMtADProofs[i] = item
		}
	}
	ChiMtAFsBzs := m.GetLTChiMtAFs()
	ChiMtAFs := make([]*big.Int, len(ChiMtAFsBzs))
	for i := range ChiMtAFs {
		Bzs := ChiMtAFsBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			ChiMtAFs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiMtADsBzs := m.GetLTChiMtADs()
	ChiMtADs := make([]*big.Int, len(ChiMtADsBzs))
	for i := range ChiMtADs {
		Bzs := ChiMtADsBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			ChiMtADs[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	ChiMtADProofsBzs := m.GetLTChiMtADProofs()
	ChiMtADProofs := make([]*zkpaffg.ProofAffg, len(ChiMtADProofsBzs)/zkpaffg.ProofAffgBytesParts)
	for i := range ChiMtADProofs {
		//if ChiMtADProofsBzs[i*zkpaffg.ProofAffgBytesParts] != nil {
		if len(ChiMtADProofsBzs[i*zkpaffg.ProofAffgBytesParts]) > 0 {
			item, err := zkpaffg.NewProofFromBytes(ec, ChiMtADProofsBzs[(i*zkpaffg.ProofAffgBytesParts):(i*zkpaffg.ProofAffgBytesParts+zkpaffg.ProofAffgBytesParts)])
			if err != nil {
				return nil, err
			}
			ChiMtADProofs[i] = item
		}
	}

	R5msgHBzs := m.GetLTr5MsgH()
	R5msgH := make([]*big.Int, len(R5msgHBzs))
	for i := range R5msgH {
		Bzs := R5msgHBzs[i]
		//if Bzs != nil {
		if len(Bzs) > 0 {
			R5msgH[i] = new(big.Int).SetBytes(Bzs)
		}
	}
	R5msgProofMulBzs := m.GetLTr5MsgProofMul()
	R5msgProofMul := make([]*zkpmul.ProofMul, len(R5msgProofMulBzs)/zkpmul.ProofMulBytesParts)
	for i := range R5msgProofMul {
		//if R5msgProofMulBzs[i*zkpmul.ProofMulBytesParts] != nil {
		if len(R5msgProofMulBzs[i*zkpmul.ProofMulBytesParts]) > 0 {
			item, err := zkpmul.NewProofFromBytes(R5msgProofMulBzs[(i * zkpmul.ProofMulBytesParts):(i*zkpmul.ProofMulBytesParts + zkpmul.ProofMulBytesParts)])
			if err != nil {
				return nil, err
			}
			R5msgProofMul[i] = item
		}
	}
	//r5msgDeltaShareEncBzs := m.GetLTr5MsgDeltaShareEnc()
	//r5msgDeltaShareEnc := make([]*big.Int, len(r5msgDeltaShareEncBzs))
	//for i := range r5msgDeltaShareEnc {
	//	Bzs := r5msgDeltaShareEncBzs[i]
	//	if Bzs != nil {
	//		r5msgDeltaShareEnc[i] = new(big.Int).SetBytes(Bzs)
	//	}
	//}
	R5msgProofDecBzs := m.GetLTr5MsgProofDec()
	R5msgProofDec := make([]*zkpdec.ProofDec, len(R5msgProofDecBzs)/zkpdec.ProofDecBytesParts)
	for i := range R5msgProofDec {
		//if R5msgProofDecBzs[i*zkpdec.ProofDecBytesParts] != nil {
		if len(R5msgProofDecBzs[i*zkpdec.ProofDecBytesParts]) > 0 {
			item, err := zkpdec.NewProofFromBytes(R5msgProofDecBzs[(i * zkpdec.ProofDecBytesParts):(i*zkpdec.ProofDecBytesParts + zkpdec.ProofDecBytesParts)])
			if err != nil {
				return nil, err
			}
			R5msgProofDec[i] = item
		}
	}
	length := len(R5msgH) // Notice, using this length
	R5msgDjisBzs := m.GetLTr5MsgDjis()
	R5msgDjis := make([][]*big.Int, length)
	for j := 0; j < length; j++ {
		R5msgDjis[j] = make([]*big.Int, length)
	}
	for i, row := range R5msgDjis {
		for j := range row {
			Bzs := R5msgDjisBzs[i*length+j]
			//if Bzs != nil {
			if len(Bzs) > 0 {
				R5msgDjis[i][j] = new(big.Int).SetBytes(Bzs)
			}
		}
	}
	R5msgFjisBzs := m.GetLTr5MsgFjis()
	R5msgFjis := make([][]*big.Int, length)
	for j := 0; j < length; j++ {
		R5msgFjis[j] = make([]*big.Int, length)
	}
	for i, row := range R5msgFjis {
		for j := range row {
			Bzs := R5msgFjisBzs[i*length+j]
			//if Bzs != nil {
			if len(Bzs) > 0 {
				R5msgFjis[i][j] = new(big.Int).SetBytes(Bzs)
			}
		}
	}
	//r5msgQ3EncBzs := m.GetLTr5MsgQ3Enc()
	//r5msgQ3Enc := make([]*big.Int, len(r5msgQ3EncBzs))
	//for i := range r5msgQ3Enc {
	//	Bzs := r5msgQ3EncBzs[i]
	//	if Bzs != nil {
	//		r5msgQ3Enc[i] = new(big.Int).SetBytes(Bzs)
	//	}
	//}

	LocalTemp := &localTempData{
		Ssid:   Ssid,
		W:      W,
		BigWs:  BigWs,
		KShare: KShare,

		BigGammaShare: BigGammaShare,
		K:             K,
		G:             G,
		KNonce:        KNonce,
		GNonce:        GNonce,

		GammaShare:      GammaShare,
		DeltaShareBetas: DeltaShareBetas,
		ChiShareBetas:   ChiShareBetas,

		BigGamma:         BigGamma,
		DeltaShareAlphas: DeltaShareAlphas,
		ChiShareAlphas:   ChiShareAlphas,
		DeltaShare:       DeltaShare,
		ChiShare:         ChiShare,
		BigDeltaShare:    BigDeltaShare,

		BigR:       BigR,
		Rx:         Rx,
		SigmaShare: SigmaShare,

		R1msgG:     R1msgG,
		R1msgK:     R1msgK,
		R1msgProof: R1msgProof,

		R2msgBigGammaShare: R2msgBigGammaShare,
		R2msgDeltaD:        R2msgDeltaD,
		R2msgDeltaF:        R2msgDeltaF,
		R2msgDeltaProof:    R2msgDeltaProof,
		R2msgChiD:          R2msgChiD,
		R2msgChiF:          R2msgChiF,
		R2msgChiProof:      R2msgChiProof,
		R2msgProofLogstar:  R2msgProofLogstar,

		R3msgDeltaShare:    R3msgDeltaShare,
		R3msgBigDeltaShare: R3msgBigDeltaShare,
		R3msgProofLogstar:  R3msgProofLogstar,

		DeltaMtAFs:      DeltaMtAFs,
		DeltaMtADs:      DeltaMtADs,
		DeltaMtADProofs: DeltaMtADProofs,
		ChiMtAFs:        ChiMtAFs,
		ChiMtADs:        ChiMtADs,
		ChiMtADProofs:   ChiMtADProofs,
		R5msgH:          R5msgH,
		R5msgProofMul:   R5msgProofMul,
		//r5msgDeltaShareEnc: r5msgDeltaShareEnc,
		R5msgProofDec: R5msgProofDec,
		R5msgDjis:     R5msgDjis,
		R5msgFjis:     R5msgFjis,
		//r5msgQ3Enc:         r5msgQ3Enc,
	}

	return LocalTemp, nil
}
