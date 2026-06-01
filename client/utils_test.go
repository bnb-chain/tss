package client

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/bnb-chain/tss-lib/v3/tss"
)

func TestCompressedPubKeySecp256k1Generator(t *testing.T) {
	expected := "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	key := ecdsa.PublicKey{
		Curve: tss.EC(),
		X:     tss.EC().Params().Gx,
		Y:     tss.EC().Params().Gy,
	}

	compressed, err := compressedPubKey(key)
	if err != nil {
		t.Fatalf("compressedPubKey returned error: %v", err)
	}
	if got := hex.EncodeToString(compressed); got != expected {
		t.Fatalf("compressed public key mismatch: got %s, want %s", got, expected)
	}
}

func TestCompressedPubKeySecp256k1OddY(t *testing.T) {
	curve := tss.EC()
	oddY := new(big.Int).Sub(curve.Params().P, curve.Params().Gy)
	key := ecdsa.PublicKey{
		Curve: curve,
		X:     curve.Params().Gx,
		Y:     oddY,
	}

	compressed, err := compressedPubKey(key)
	if err != nil {
		t.Fatalf("compressedPubKey returned error: %v", err)
	}
	if compressed[0] != 0x03 {
		t.Fatalf("compressed public key prefix mismatch: got 0x%02x, want 0x03", compressed[0])
	}
}

func TestCompressedPubKeyRejectsIncompleteKey(t *testing.T) {
	testCases := []struct {
		name string
		key  ecdsa.PublicKey
	}{
		{
			name: "missing curve",
			key: ecdsa.PublicKey{
				X: big.NewInt(1),
				Y: big.NewInt(2),
			},
		},
		{
			name: "missing x",
			key: ecdsa.PublicKey{
				Curve: tss.EC(),
				Y:     big.NewInt(2),
			},
		},
		{
			name: "missing y",
			key: ecdsa.PublicKey{
				Curve: tss.EC(),
				X:     big.NewInt(1),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := compressedPubKey(tc.key); err == nil {
				t.Fatal("compressedPubKey returned nil error")
			}
		})
	}
}
