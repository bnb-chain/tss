package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/bgentry/speakeasy"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(keygenCmd)
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "key generation",
	Long:  "generate secret share of t of n scheme",
	PreRun: func(cmd *cobra.Command, args []string) {
		common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		checkBootstrap(cmd, args)
		checkN()
		setT()
		setPeers()
		setPassphrase()
		updateConfig()

		c := client.NewTssClient(common.TssCfg, client.KeygenMode, false)
		c.Start()
	},
}

func checkBootstrap(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	answer, err := GetString("Do you like re-bootstrap again?[y/N]: ", reader)
	if err != nil {
		panic(err)
	}
	if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
		bootstrap.Run(cmd, args)
		common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
	}
}

func checkN() {
	if common.TssCfg.Parties > 0 && len(common.TssCfg.ExpectedPeers) != common.TssCfg.Parties-1 {
		panic("peers are not correctly set during bootstrap")
	}
}

func setT() {
	if common.TssCfg.Threshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := GetInt("please set threshold(t), at least t + 1 parties needs participant signing: ", reader)
	if err != nil {
		panic(err)
	}
	if t <= 0 {
		panic(fmt.Errorf("t should greater than 0"))
	}
	if t+1 >= common.TssCfg.Parties {
		panic(fmt.Errorf("t + 1 should less than parties"))
	}
	common.TssCfg.Threshold = t
}

func setPeers() {
	if len(common.TssCfg.ExpectedPeers) == common.TssCfg.Parties-1 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	peers := make([]string, 0, common.TssCfg.Parties-1)
	peer_addrs := make([]string, 0, common.TssCfg.Parties-1)
	needPeerAddr := len(common.TssCfg.BootstrapPeers) == 0 && !common.TssCfg.DefaultBootstap
	for i := 1; i < common.TssCfg.Parties; i++ {
		ithParty := humanize.Ordinal(i)
		moniker, err := GetString(fmt.Sprintf("please input moniker of the %s party", ithParty), reader)
		if err != nil {
			panic(err)
		}
		id, err := GetString(fmt.Sprintf("please input id of the %s party", ithParty), reader)
		peers = append(peers, fmt.Sprintf("%s@%s", moniker, id))

		if needPeerAddr {
			addr, err := GetString(fmt.Sprintf("please input peer listen address of the %s party (e.g. /ip4/127.0.0.1/tcp/27148)", ithParty), reader)
			if err != nil {
				panic(err)
			}
			peer_addrs = append(peer_addrs, addr)
		}
	}
	common.TssCfg.ExpectedPeers = peers
	common.TssCfg.PeerAddrs = peer_addrs
}

func setPassphrase() {
	if common.TssCfg.Password != "" {
		return
	}

	if p, err := speakeasy.Ask("please input password to secure secret key:"); err == nil {
		if p2, err := speakeasy.Ask("please input again:"); err == nil {
			if p2 != p {
				panic(fmt.Errorf("two inputs does not match, please start again"))
			} else {
				checkComplexityOfPassword(p)
				common.TssCfg.Password = p
			}
		} else {
			panic(err)
		}
	} else {
		panic(err)
	}
}

func checkComplexityOfPassword(p string) {
	if len(p) <= 8 {
		panic(fmt.Errorf("password is too simple, should be longer than 8 characters"))
	}
}

func updateConfig() {
	bytes, err := json.MarshalIndent(&common.TssCfg, "", "    ")
	if err != nil {
		panic(err)
	}
	if err = ioutil.WriteFile(path.Join(common.TssCfg.Home, "config.json"), bytes, os.FileMode(0600)); err != nil {
		panic(err)
	}
}
