package cmd

import (
	"os"
	"path"

	"github.com/ipfs/go-log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

const (
	flagHome            = "home"
	flagVault           = "vault_name"
	flagPassword        = "password"
	flagLogLevel        = "log_level"
	flagPrefix          = "address_prefix"
	flagMoniker         = "moniker"
	flagThreshold       = "threshold"
	flagParties         = "parties"
	flagChannelId       = "channel_id"
	flagChannelPassword = "channel_password"
	flagBroadcastSanityCheck = "p2p.broadcast_sanity_check"
)

var rootCmd = &cobra.Command{
	Use:   "tss",
	Short: "Threshold signing scheme",
	Long:  `Complete documentation is available at https://github.com/binance-chain/tss`, // TODO: replace documentation here
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	initConfigAndLogLevel()

	if err := rootCmd.Execute(); err != nil {
		client.Logger.Error(err)
		os.Exit(1)
	}
}

func initConfigAndLogLevel() {
	bindP2pConfigs()

	home, err := os.UserHomeDir()
	if err != nil {
		common.Panic(err)
	}
	rootCmd.PersistentFlags().String(flagHome, path.Join(home, ".tss"), "Path to config/route_table/node_key/tss_key files, configs in config file can be overridden by command line arg quments")
	rootCmd.PersistentFlags().String(flagVault, "", "name of vault of this party")
	rootCmd.PersistentFlags().String(flagPassword, "", "password, should only be used for testing. If empty, you will be prompted for password to save/load the secret/public share and config")
	rootCmd.PersistentFlags().String(flagLogLevel, "info", "log level")
}

func bindP2pConfigs() {
	initCmd.PersistentFlags().String("p2p.listen", "", "Adds a multiaddress to the listen list")
	//rootCmd.PersistentFlags().StringSlice("p2p.bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	//rootCmd.PersistentFlags().StringSlice("p2p.relays", []string{}, "relay server list")
	keygenCmd.PersistentFlags().StringSlice("p2p.peer_addrs", []string{}, "peer's multiple addresses")
	regroupCmd.PersistentFlags().StringSlice("p2p.new_peer_addrs", []string{}, "unknown peer's multiple addresses")
	//rootCmd.PersistentFlags().StringSlice("p2p.peers", []string{}, "peers in this threshold scheme")
	//rootCmd.PersistentFlags().Bool("p2p.default_bootstrap", false, "whether to use default bootstrap")
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)
	log.SetLogLevel("common", cfg.LogLevel)
	log.SetLogLevel("blockchain", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
	log.SetLogLevel("stream-upgrader", "error")
}
