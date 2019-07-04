package client

import (
	"crypto/elliptic"
	"math/big"

	"github.com/binance-chain/tss-lib/signing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/tendermint/tendermint/crypto"
)

// This file is bridging TssClient with tendermint PrivKey interface
// So that TssClient can be used as PrivKey for cosmos keybase

func (*TssClient) Bytes() []byte {
	return []byte("HAHA, we do not know private key")
}

func (client *TssClient) Sign(msg []byte) ([]byte, error) {
	hash := crypto.Sha256(msg)
	m := hashToInt(hash, tss.EC())
	return client.signImpl(m)
}

func (client *TssClient) PubKey() crypto.PubKey {
	if pubKey, err := LoadPubkey(client.config.Home); err == nil {
		return pubKey
	} else {
		return nil
	}
}

func (*TssClient) Equals(key crypto.PrivKey) bool {
	return true
}

func (client *TssClient) signImpl(m *big.Int) ([]byte, error) {
	logger.Infof("[%s] message to be signed: %s\n", client.config.Moniker, m.String())
	client.localParty = signing.NewLocalParty(m, client.params, *client.key, client.sendCh, client.signCh)

	// has to start local party before network routines in case 2 other peers' msg comes before self fully initialized
	if err := client.localParty.Start(); err != nil {
		panic(err)
	}

	done := make(chan bool)
	go client.sendMessageRoutine(client.sendCh)
	go client.handleMessageRoutine()
	go client.saveSignatureRoutine(client.signCh, done)

	<-done
	logger.Debugf("[%s] received signature: %X", client.config.Moniker, client.signature)
	return client.signature, nil
}

// copied from https://github.com/btcsuite/btcd/blob/c26ffa870fd817666a857af1bf6498fabba1ffe3/btcec/signature.go#L263
func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
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
