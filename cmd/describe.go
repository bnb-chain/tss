package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
)

func init() {
	describeCmd.PersistentFlags().String(flagPrefix, "bnb", "prefix of bech32 address")
	describeCmd.PersistentFlags().String(flagNetwork, "Binance", "")

	rootCmd.AddCommand(describeCmd)
}

// fmt.Printf is deliberately used in this command
var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "show config and address of a tss vault",
	Long:  "",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var from string
		network := viper.GetString(flagNetwork)
		if strings.HasPrefix(network, "Binance") {
			_, from = initBinance(network)
		} else if strings.HasPrefix(network, "Ethereum") {
			_, from = initEthereum(network)
		}
		fmt.Printf("address of this vault: %s\n", from)

		cfg, err := json.MarshalIndent(common.TssCfg, "", "\t")
		if err != nil {
			common.Panic(err)
		}
		fmt.Printf("config of this vault:\n%s\n", string(cfg))
	},
}
