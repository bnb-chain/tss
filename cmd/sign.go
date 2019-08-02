package cmd

import (
	"fmt"

	"github.com/bgentry/speakeasy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign a transaction",
	Long:  "sign a transaction using local share, signers will be prompted to fill in",
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home")); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		askPassphrase()
		setChannelId()
		setChannelPasswd()
		setMessage()

		c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
		c.Start()
	},
}

func askPassphrase() {
	if common.TssCfg.Password != "" {
		return
	}

	if p, err := speakeasy.Ask(fmt.Sprintf("Password to sign with '%s':", common.TssCfg.Moniker)); err == nil {
		common.TssCfg.Password = p
	} else {
		panic(err)
	}
}

// TODO: use MessageBridge
func setMessage() {
	common.TssCfg.Message = "0"
}
