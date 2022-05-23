package cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Safulet/tss/client"
	"github.com/Safulet/tss/common"
)

func init() {
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
		signRun()
	},
}

func signRun() {
	setChannelId()
	setChannelPasswd()
	setMessage()
	common.PrintPrefixed(common.TssCfg.Message)
	setFromMobile()

	c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
	text, _ := json.Marshal(common.TssCfg)
	common.PrintPrefixed(string(text))
	c.Start()
}

// TODO: use MessageBridge
func setMessage() {
	if common.TssCfg.Message == "" {
		common.TssCfg.Message = "0"
	}
}

func setFromMobile() {
	common.TssCfg.FromMobile = viper.GetBool("from_mobile")
}
