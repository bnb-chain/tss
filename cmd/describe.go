package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(describeCmd)
}

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "show config and address of a tss vault",
	Long:  "",
	PreRun: func(cmd *cobra.Command, args []string) {
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString(flagHome), viper.GetString(flagVault), passphrase); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		pubKey, err := common.LoadEcdsaPubkey(viper.GetString(flagHome), viper.GetString(flagVault), common.TssCfg.Password)
		if err != nil {
			panic(err)
		}
		addr, err := client.GetAddress(*pubKey, viper.GetString(flagPrefix))
		if err != nil {
			panic(err)
		}
		fmt.Printf("address of this vault: %s\n", addr)
		cfg, err := json.MarshalIndent(common.TssCfg, "", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Printf("config of this vault:\n%s\n", string(cfg))
	},
}
