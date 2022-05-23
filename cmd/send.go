package cmd

import (
	"github.com/Safulet/chain-integration/v2/blockchain"
	"github.com/Safulet/tss/client"
	"github.com/Safulet/tss/common"
	"github.com/Safulet/tss/utils/blockchainUtils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whyrusleeping/go-logging"
	"math/big"
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "send a transaction",
	Long:  "send a transaction",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		askChannel()
		askPassphrase()
		err := checkSendParameters()
		if err != nil {
			common.Panic(err)
		}
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		common.TssCfg.LogLevel = logging.INFO.String() // no configs here, mandatory set log level
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		transfer := buildSendTransfer()
		// verify to address
		err := blockchainUtils.ValidateToAddress(transfer.Network, transfer.To, true)
		if err != nil {
			common.Panic(err)
		}
		blockchianSupport, err := buildBlockchianSupport(&transfer, viper.GetString("pubKeyHex"))
		if err != nil {
			common.Panic(err)
		}

		var gasPrice *big.Int
		if transfer.GetGasPrice() != "" {
			var ok bool
			gasPrice, ok = new(big.Int).SetString(transfer.GetGasPrice(), 10)
			if !ok {
				common.Panic(errors.New("can not pass gas price"))
			}
		}

		messages, _, _, err := blockchianSupport.BuildPreImage(blockchain.BuildPreImageData{
			Amount:   amountWithDecimal(&transfer),
			From:     transfer.From,
			To:       transfer.To,
			Asset:    transfer.Asset,
			GasPrice: gasPrice,
		})

		if err != nil {
			common.Panic(err)
		}

		var signatureMessage common.SignatureMessage
		for _, msg := range messages {
			common.TssCfg.Message = new(big.Int).SetBytes(msg).String()
			signRun()

			if err != nil {
				common.Panic(err)
			}

			signatureMessage.Signatures = append(signatureMessage.Signatures, &common.SignatureMessage_SignatureData{
				R:                 common.Signature.R,
				S:                 common.Signature.S,
				M:                 common.Signature.M,
				Signature:         common.Signature.Signature,
				SignatureRecovery: common.Signature.SignatureRecovery,
			})
		}

		hash, err := Broadcast(blockchianSupport, &signatureMessage)
		if err != nil {
			common.Panic(err)
		} else {
			client.Logger.Infof("transaction submitted, the hash is %v", hash)
		}
	},
}

func buildBlockchianSupport(transfer *common.Transfer, pubkeyAddr string) (blockchain.Blockchain, error) {
	return blockchainUtils.InitBlockchain(transfer.Network, transfer.From, pubkeyAddr, true)
}

func buildSendTransfer() common.Transfer {
	from := viper.GetString("from")
	to := viper.GetString("to")
	network := viper.GetString("network")
	asset := viper.GetString("asset")
	amount := viper.GetString("amount")
	gasPrice := viper.GetString("gasPrice")
	nonce := viper.GetInt64("nonce")

	return generateTransfer(network, from, to, amount, asset, gasPrice, nonce)
}

func checkSendParameters() error {
	if viper.GetString("from") == "" {
		return errors.New("can not get valid from")
	}
	if viper.GetString("to") == "" {
		return errors.New("can not get valid to")
	}
	if viper.GetString("network") == "" {
		return errors.New("can not get valid network")
	}
	if viper.GetString("asset") == "" {
		return errors.New("can not get valid asset")
	}
	if viper.GetString("amount") == "" {
		return errors.New("can not get valid amount")
	}
	if viper.GetString("pubKeyHex") == "" {
		return errors.New("can not get valid pubKeyHex")
	}
	return nil
}
