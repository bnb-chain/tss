// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpmulstar

import (
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
)

const (
	ProofMulstarBytesParts = 8
)

type (
	ProofMulstar struct {
		A, Bx, By, E, S, Z1, Z2, W *big.Int
	}
)

// NewProof implements proofmulstar
func NewProof(Session []byte, ec elliptic.Curve, pk *paillier.PublicKey, g, X *crypto.ECPoint, C, D, NCap, s, t, x, rho *big.Int) (*ProofMulstar, error) {
	if ec == nil || pk == nil || g == nil || X == nil || C == nil || D == nil || NCap == nil || s == nil || t == nil || x == nil || rho == nil {
		return nil, errors.New("ProveMulstar constructor received nil value(s)")
	}
	q := ec.Params().N
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)
	qNCap := new(big.Int).Mul(q, NCap)
	q3NCap := new(big.Int).Mul(q3, NCap)

	// Fig 31.1 sample
	alpha := common.GetRandomPositiveInt(q3)
	r := common.GetRandomPositiveRelativelyPrimeInt(pk.N)
	gamma := common.GetRandomPositiveInt(q3NCap)
	m := common.GetRandomPositiveInt(qNCap)

	// Fig 31.1 compute
	modN2 := common.ModInt(pk.NSquare())
	A := modN2.Exp(C, alpha)
	A = modN2.Mul(A, modN2.Exp(r, pk.N))

	B := g.ScalarMult(alpha)

	modNCap := common.ModInt(NCap)
	E := modNCap.Exp(s, alpha)
	E = modNCap.Mul(E, modNCap.Exp(t, gamma))

	S := modNCap.Exp(s, x)
	S = modNCap.Mul(S, modNCap.Exp(t, m))

	// Fig 31.2 e
	var e *big.Int
	{
		eHash := common.SHA512_256i_TAGGED(Session, append(pk.AsInts(), ec.Params().B, ec.Params().N, ec.Params().P, g.X(), g.Y(), X.X(), X.Y(), C, D, NCap, s, t, A, B.X(), B.Y(), E, S)...)
		e = common.RejectionSample(q, eHash)
	}

	// Fig 31.3 reply
	z1 := new(big.Int).Mul(e, x)
	z1 = new(big.Int).Add(z1, alpha)

	z2 := new(big.Int).Mul(e, m)
	z2 = new(big.Int).Add(z2, gamma)

	modN := common.ModInt(pk.N)
	w := modN.Exp(rho, e)
	w = modN.Mul(w, r)

	return &ProofMulstar{A: A, Bx: B.X(), By: B.Y(), E: E, S: S, Z1: z1, Z2: z2, W: w}, nil
}

func NewProofFromBytes(bzs [][]byte) (*ProofMulstar, error) {
	if !common.NonEmptyMultiBytes(bzs, ProofMulstarBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofMulstar", ProofMulstarBytesParts)
	}
	return &ProofMulstar{
		A:  new(big.Int).SetBytes(bzs[0]),
		Bx: new(big.Int).SetBytes(bzs[1]),
		By: new(big.Int).SetBytes(bzs[2]),
		E:  new(big.Int).SetBytes(bzs[3]),
		S:  new(big.Int).SetBytes(bzs[4]),
		Z1: new(big.Int).SetBytes(bzs[5]),
		Z2: new(big.Int).SetBytes(bzs[6]),
		W:  new(big.Int).SetBytes(bzs[7]),
	}, nil
}

func (pf *ProofMulstar) Verify(Session []byte, ec elliptic.Curve, pk *paillier.PublicKey, g, X *crypto.ECPoint, C, D, NCap, s, t *big.Int) bool {
	if ec == nil || pk == nil || g == nil || X == nil || C == nil || D == nil || NCap == nil || s == nil || t == nil {
		return false
	}

	q := ec.Params().N
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)

	var e *big.Int
	{
		eHash := common.SHA512_256i_TAGGED(Session, append(pk.AsInts(), ec.Params().B, ec.Params().N, ec.Params().P, g.X(), g.Y(), X.X(), X.Y(), C, D, NCap, s, t, pf.A, pf.Bx, pf.By, pf.E, pf.S)...)
		e = common.RejectionSample(q, eHash)
	}

	// Fig 31. Equality Check
	{
		modN2 := common.ModInt(pk.NSquare())
		LHS := modN2.Mul(modN2.Exp(C, pf.Z1), modN2.Exp(pf.W, pk.N))
		RHS := modN2.Mul(pf.A, modN2.Exp(D, e))

		if LHS.Cmp(RHS) != 0 {
			return false
		}
	}

	{
		LHS := g.ScalarMult(pf.Z1)
		// left := crypto.ScalarBaseMult(ec, z1ModQ)
		B := crypto.NewECPointNoCurveCheck(ec, pf.Bx, pf.By)
		RHS, err := X.ScalarMult(e).Add(B)
		if err != nil || !LHS.Equals(RHS) {
			return false
		}
	}

	{
		modNCap := common.ModInt(NCap)
		LHS := modNCap.Mul(modNCap.Exp(s, pf.Z1), modNCap.Exp(t, pf.Z2))
		RHS := modNCap.Mul(pf.E, modNCap.Exp(pf.S, e))

		if LHS.Cmp(RHS) != 0 {
			return false
		}
	}
	return true
}

func (pf *ProofMulstar) ValidateBasic() bool {
	return pf.A != nil &&
		pf.Bx != nil &&
		pf.By != nil &&
		pf.E != nil &&
		pf.S != nil &&
		pf.Z1 != nil &&
		pf.Z2 != nil &&
		pf.W != nil
}

func (pf *ProofMulstar) Bytes() [ProofMulstarBytesParts][]byte {
	return [...][]byte{
		pf.A.Bytes(),
		pf.Bx.Bytes(),
		pf.By.Bytes(),
		pf.E.Bytes(),
		pf.S.Bytes(),
		pf.Z1.Bytes(),
		pf.Z2.Bytes(),
		pf.W.Bytes(),
	}
}
