package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
		common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
	},
}
