package client

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"math/big"
	"os"
	"path"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/bech32"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"

	"github.com/binance-chain/tss/common"
)

func loadSavedKey(config *common.TssConfig) keygen.LocalPartySaveData {
	wPriv, err := os.OpenFile(path.Join(config.Home, config.Vault, "sk.json"), os.O_RDONLY, 0400)
	if err != nil {
		common.Panic(err)
	}
	defer wPriv.Close()
	wPub, err := os.OpenFile(path.Join(config.Home, config.Vault, "pk.json"), os.O_RDONLY, 0400)
	if err != nil {
		common.Panic(err)
	}
	defer wPub.Close()

	result, _, err := common.Load(config.Password, wPriv, wPub) // TODO: validate nodeKey
	if err != nil {
		common.Panic(err)
	}
	return *result
}

func newEmptySaveData() keygen.LocalPartySaveData {
	return keygen.LocalPartySaveData{
		BigXj:       make([]*crypto.ECPoint, common.TssCfg.NewParties),
		PaillierPKs: make([]*paillier.PublicKey, common.TssCfg.NewParties),
		NTildej:     make([]*big.Int, common.TssCfg.NewParties),
		H1j:         make([]*big.Int, common.TssCfg.NewParties),
		H2j:         make([]*big.Int, common.TssCfg.NewParties),
	}
}

func appendIfNotExist(target []string, new string) []string {
	exist := false
	for _, old := range target {
		if old == new {
			exist = true
			break
		}
	}
	if !exist {
		target = append(target, new)
	}
	return target
}

func GetAddress(key ecdsa.PublicKey, prefix string) (string, error) {
	btcecPubKey := btcec.PublicKey(key)
	// be consistent with tendermint/crypto
	compressed := btcecPubKey.SerializeCompressed()
	hasherSHA256 := sha256.New()
	hasherSHA256.Write(compressed[:]) // does not error
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error

	address := []byte(hasherRIPEMD160.Sum(nil))
	converted, err := bech32.ConvertBits(address, 8, 5, true) // TODO: error check
	if err != nil {
		return "", errors.Wrap(err, "encoding bech32 failed")
	}
	return bech32.Encode(prefix, converted)
}
