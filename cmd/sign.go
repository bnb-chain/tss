// +build deluxe

package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/binance-chain/tss/blockchain"
	"github.com/binance-chain/tss/blockchain/binance"
	"github.com/binance-chain/tss/blockchain/ethereum"
	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

const (
	flagNetwork   = "network"
	flagTo        = "to"
	flagAmount    = "amount"
	flagDemon     = "demon"
	flagBroadcast = "broadcast"
)

func init() {
	signCmd.PersistentFlags().String(flagChannelId, "", "channel id of this session")
	signCmd.PersistentFlags().String(flagChannelPassword, "", "channel password of this session")
	signCmd.PersistentFlags().Bool(flagBroadcastSanityCheck, true, "whether verify broadcast message's hash with peers")

	signCmd.PersistentFlags().String(flagNetwork, "", "")
	signCmd.PersistentFlags().String(flagTo, "", "to address")
	signCmd.PersistentFlags().Float64(flagAmount, 0, "amount")
	signCmd.PersistentFlags().String(flagDemon, "", "demon")
	signCmd.PersistentFlags().Bool(flagBroadcast, false, "broadcast the transcation to blockchain")

	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign a transaction",
	Long:  "sign a transaction using local share, signers will be prompted to fill in",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		setChannelId()
		setChannelPasswd()

		var blockchain blockchain.AccountBlockchain
		var from string
		network := viper.GetString(flagNetwork)
		if strings.HasPrefix(network, "Binance") {
			blockchain, from = initBinance(network)
		} else if strings.HasPrefix(network, "Ethereum") {
			blockchain, from = initEthereum(network)
		}

		preImages, err := blockchain.BuildPreImage(int64(viper.GetFloat64(flagAmount)), from, viper.GetString(flagTo), viper.GetString(flagDemon))
		if err != nil {
			panic(err)
		}
		msgBuilder := strings.Builder{}
		for _, preImage := range preImages {
			fmt.Fprint(&msgBuilder, hex.EncodeToString(preImage))
		}
		common.TssCfg.Message = msgBuilder.String()
		c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
		signatures, err := c.SignImpl(preImages)
		if err != nil {
			panic(err)
		}

		transaction, err := blockchain.BuildTransaction(signatures)
		if err != nil {
			panic(err)
		}
		if viper.GetBool(flagBroadcast) {
			txHash, err := blockchain.Broadcast(transaction)
			if err != nil {
				panic(err)
			}
			fmt.Println(hex.EncodeToString(txHash))
		}

		return
	},
}

func initBinance(network string) (blockchain.AccountBlockchain, string) {
	pubkey, err := common.LoadEcdsaPubkey(common.TssCfg.Home, common.TssCfg.Vault, common.TssCfg.Password)
	if err != nil {
		panic(err)
	}
	btcecPubKey := (*btcec.PublicKey)(pubkey)
	var pubkeyBytes secp256k1.PubKeySecp256k1
	copy(pubkeyBytes[:], btcecPubKey.SerializeCompressed())

	var blockchain blockchain.AccountBlockchain
	if network == "Binance" {
		blockchain = &binance.Binance{PublicKey: pubkeyBytes, Network: binance.Mainnet}
	} else if network == "BinanceTestnet" {
		blockchain = &binance.Binance{PublicKey: pubkeyBytes, Network: binance.Testnet}
	} else {
		panic(fmt.Errorf("network %s is not supported", network))
	}

	from, err := blockchain.GetAddress(pubkeyBytes[:])
	if err != nil {
		panic(err)
	}
	return blockchain, from
}

func initEthereum(network string) (blockchain.AccountBlockchain, string) {
	ecdsaPubKey, err := common.LoadEcdsaPubkey(common.TssCfg.Home, common.TssCfg.Vault, common.TssCfg.Password)
	if err != nil {
		panic(err)
	}

	btcecPubKey := (*btcec.PublicKey)(ecdsaPubKey)
	pubkeyBytes := make([]byte, btcec.PubKeyBytesLenUncompressed, btcec.PubKeyBytesLenUncompressed)
	copy(pubkeyBytes[:], btcecPubKey.SerializeUncompressed())

	var blockchain blockchain.AccountBlockchain
	if network == "Ethereum" {
		blockchain = &ethereum.Ethereum{Network: ethereum.Mainnet}
	} else if network == "EthereumRopsten" {
		blockchain = &ethereum.Ethereum{Network: ethereum.Ropsten}
	} else {
		panic(fmt.Errorf("network %s is not supported", network))
	}

	from, err := blockchain.GetAddress(pubkeyBytes)
	if err != nil {
		panic(err)
	}
	return blockchain, from
}
