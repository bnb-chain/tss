package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/server"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "bootstrap and relay server helps node (dynamic ip) discovery and NAT traversal",
	Long:  "bootstrap and relay server helps node (dynamic ip) discovery and NAT traversal",
	Run: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
		server.NewTssP2PServer(common.TssCfg.Home, common.TssCfg.Vault, common.TssCfg.P2PConfig)
		select {}
	},
}
