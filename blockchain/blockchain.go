package blockchain

import (
	"net/http"
	"time"

	"github.com/ipfs/go-log"
)

var logger = log.Logger("blockchain")

func init() {
	// TODO: switch to resty lib
	http.DefaultClient.Timeout = 10 * time.Second
}

// common interface for different account based (rather than UTXO) block chains
type AccountBlockchain interface {
	// human readable address of this blockchain
	GetAddress(publicKey []byte) (string, error)
	// build message to be signed
	BuildPreImage(amount int64, from, to, demon string) ([][]byte, error)
	// build transaction to be broadcast
	// TODO: this implementation is coupled with 65 bytse ecdsa signature (sig + 1 byte recover byte)
	BuildTransaction(signatures [][]byte) ([]byte, error)
	// broadcast transaction to blockchain node, the transaction hash is returned
	Broadcast(transaction []byte) ([]byte, error)
}
