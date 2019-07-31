package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/ipfs/go-log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfigAndLogLevel() {
	bindP2pConfigs()
	bindKdfConfigs()
	bindClientConfigs()

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().String("home", path.Join(home, ".tss"), "Path to config/route_table/node_key/tss_key files, configs in config file can be overriden by command line arguments")
}

func bindP2pConfigs() {
	rootCmd.PersistentFlags().String("p2p.listen", "", "Adds a multiaddress to the listen list")
	rootCmd.PersistentFlags().String("p2p.log_level", "info", "log level")
	rootCmd.PersistentFlags().StringSlice("p2p.bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	rootCmd.PersistentFlags().StringSlice("p2p.relays", []string{}, "relay server list")
	rootCmd.PersistentFlags().StringSlice("p2p.peer_addrs", []string{}, "peer's multiple addresses")
	rootCmd.PersistentFlags().StringSlice("p2p.peers", []string{}, "peers in this threshold scheme")
	rootCmd.PersistentFlags().Bool("p2p.default_bootstrap", false, "whether to use default bootstrap")
	rootCmd.PersistentFlags().Bool("p2p.broadcast_sanity_check", true, "whether verify broadcast message's hash with peers")
}

// more detail explaination of these parameters can be found:
// https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
// https://www.alexedwards.net/blog/how-to-hash-and-verify-passwords-with-argon2-in-go
func bindKdfConfigs() {
	rootCmd.PersistentFlags().Uint32("kdf.memory", 65536, "The amount of memory used by the algorithm (in kibibytes)")
	rootCmd.PersistentFlags().Uint32("kdf.iterations", 13, "The number of iterations (or passes) over the memory.")
	rootCmd.PersistentFlags().Uint8("kdf.parallelism", 4, "The number of threads (or lanes) used by the algorithm.")
	rootCmd.PersistentFlags().Uint32("kdf.salt_length", 16, "Length of the random salt. 16 bytes is recommended for password hashing.")
	rootCmd.PersistentFlags().Uint32("kdf.key_length", 48, "Length of the generated key (or password hash). must be 32 bytes or more")
}

func bindClientConfigs() {
	rootCmd.PersistentFlags().String("id", "", "id of current node")
	rootCmd.PersistentFlags().String("moniker", "", "moniker of current node")
	rootCmd.PersistentFlags().Int("threshold", 0, "threshold of this scheme")
	rootCmd.PersistentFlags().Int("parties", 0, "total parities of this scheme")
	rootCmd.PersistentFlags().Int("new_threshold", 0, "new threshold of regrouped scheme")
	rootCmd.PersistentFlags().Int("new_parties", 0, "new total parties of regrouped scheme")
	rootCmd.PersistentFlags().String("profile_addr", "", "host:port of go pprof")
	rootCmd.PersistentFlags().String("password", "", "password, should only be used for testing. If empty, you will be prompted for password to save/load the secret share")
	rootCmd.PersistentFlags().String("message", "", "message(in *big.Int.String() format) to be signed, only used in sign mode")
	rootCmd.PersistentFlags().String("channel_id", "", "channel id of this session")
	rootCmd.PersistentFlags().String("channel_password", "", "channel password of this session")

	rootCmd.PersistentFlags().Bool("silent", false, "whether to make user interactively input properties, used when properties are set in config file")
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("tss-lib", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("srv", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("trans", cfg.P2PConfig.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.P2PConfig.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
}
