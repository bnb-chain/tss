package cmd

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/Safulet/tss/client"
	"github.com/Safulet/tss/common"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/big"
	"os"
)

func init() {
	rootCmd.AddCommand(privCmd)
}

var privCmd = &cobra.Command{
	Use:   "priv",
	Short: "get priv and store",
	Long:  "get priv and store",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.Set("threshold", 1)
		common.TssCfg.Threshold = 1
		viper.Set("parties", 3)
		askThreshold()
		askParties()
	},
	Run: func(cmd *cobra.Command, args []string) {
		sum := askXi(common.TssCfg.Threshold)
		common.PrintPrefixed(fmt.Sprintf("%s", sum))
		pubKey, err := askPubKeyHex()
		if err != nil {
			common.Panic(err)
		}
		privKey, err := parsePrivKey(sum, pubKey)
		common.PrintPrefixed(fmt.Sprintf("privKey is %s, pubkey is %s", privKey, pubKey))
		signBytes, _ := privKey.Sign([]byte{0x1})
		result, _ := pubKey.Verify([]byte{0x1}, signBytes)
		common.PrintPrefixed(fmt.Sprintf("result is %v", result))
	},
}

func askXi(threshold int) *big.Int {

	reader := bufio.NewReader(os.Stdin)
	sum := new(big.Int).SetInt64(0)
	for i := 0; i < threshold+1; i++ {
		x, _ := common.GetString("Please input the value of xi", reader)
		client.Logger.Infof(x)
		xi, _ := new(big.Int).SetString(x, 10)
		sum = sum.Add(sum, xi)
	}

	return sum
}

func askPubKeyHex() (*crypto.Secp256k1PublicKey, error) {
	reader := bufio.NewReader(os.Stdin)
	pubKeyHex, _ := common.GetString("Please input the hex of pub key", reader)
	keys, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}
	pubKey, err := crypto.UnmarshalSecp256k1PublicKey(keys)
	if err != nil {
		return nil, err
	}
	pubKeyEcdsa := pubKey.(*crypto.Secp256k1PublicKey)
	return pubKeyEcdsa, nil
}

func parsePrivKey(i *big.Int, pubKey *crypto.Secp256k1PublicKey) (*crypto.Secp256k1PrivateKey, error) {
	privKey := crypto.Secp256k1PrivateKey{*(*ecdsa.PublicKey)(pubKey), i}
	return &privKey, nil
}
