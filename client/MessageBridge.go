package client

import (
	"math/big"

	"github.com/binance-chain/tss-lib/tss"
	"github.com/tendermint/tendermint/crypto"
)

var Bridges map[string]MessageBridge

// MesasgeBridge convert tx/msg to be signed into big.Int which is accepted by tss signing scheme
// differet blockchain might have different way to convert
// byte array to big.Int
// This method makes tss client can be used as standalone app to sign messages for different chain, rather than
// coupled with different cli, i.e. bnbcli
type MessageBridge func([]byte) *big.Int

func BinanceChainBridge(message []byte) *big.Int {
	hash := crypto.Sha256(message)
	return hashToInt(hash, tss.EC())
}

func init() {
	Bridges = make(map[string]MessageBridge)
	Bridges["binance-chain"] = BinanceChainBridge
}
