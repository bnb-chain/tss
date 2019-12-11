package client

import (
	"crypto/elliptic"
	"math/big"

	"github.com/bgentry/speakeasy"
	"github.com/btcsuite/btcd/btcec"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/binance-chain/tss/common"
)

// This file is bridging TssClient with tendermint PrivKey interface
// So that TssClient can be used as PrivKey for cosmos keybase

func (*TssClient) Bytes() []byte {
	return []byte("HAHA, we do not know private key")
}

func (client *TssClient) Sign(msg []byte) ([]byte, error) {
	hash := crypto.Sha256(msg)
	signatures, err := client.SignImpl([][]byte{hash})
	return signatures[0].Signature, err
}

func (client *TssClient) PubKey() crypto.PubKey {
	if pubKey, err := LoadPubkey(client.config.Home, client.config.Vault); err == nil {
		return pubKey
	} else {
		return nil
	}
}

func (*TssClient) Equals(key crypto.PrivKey) bool {
	return true
}

// This helper method is used by PubKey interface in keys.go
func LoadPubkey(home, vault string) (crypto.PubKey, error) {
	passphrase := common.TssCfg.Password
	if passphrase == "" {
		if p, err := speakeasy.Ask("> Password to sign with this vault:"); err == nil {
			passphrase = p
		} else {
			return nil, err
		}
	}

	ecdsaPubKey, err := common.LoadEcdsaPubkey(home, vault, passphrase)
	if err != nil {
		return nil, err
	}
	btcecPubKey := (*btcec.PublicKey)(ecdsaPubKey)

	var pubkeyBytes secp256k1.PubKeySecp256k1
	copy(pubkeyBytes[:], btcecPubKey.SerializeCompressed())
	return pubkeyBytes, nil
}

// copied from https://github.com/btcsuite/btcd/blob/c26ffa870fd817666a857af1bf6498fabba1ffe3/btcec/signature.go#L263
func HashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}
