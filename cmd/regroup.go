package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(regroupCmd)
}

var regroupCmd = &cobra.Command{
	Use:   "regroup",
	Short: "regroup a new set of parties and threshold",
	Long:  "generate new_n secrete share with new_t threshold. At least old_t + 1 should participant",
	PreRun: func(cmd *cobra.Command, args []string) {
		common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var isNewParty bool
		if _, err := os.Stat(path.Join(common.TssCfg.Home, "sk.json")); os.IsNotExist(err) {
			isNewParty = true
		}
		if isNewParty {
			setOldN()
			setOldT()
		}
		setNewN()
		setNewT()
		setOldParties(isNewParty)
		setNewParties()
		if isNewParty {
			setPassphrase()
		}
		updateConfig()

		c := client.NewTssClient(common.TssCfg, client.RegroupMode, false)
		c.Start()
	},
}

func setOldN() {
	if common.TssCfg.Parties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := GetInt("please set old total parties(n): ", reader)
	if err != nil {
		panic(err)
	}
	if n <= 1 {
		panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.Parties = n
}

func setOldT() {
	if common.TssCfg.Threshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := GetInt("please set old threshold(t), at least t + 1 parties needs participant signing: ", reader)
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

func setNewN() {
	if common.TssCfg.NewParties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := GetInt("please set new total parties(n): ", reader)
	if err != nil {
		panic(err)
	}
	if n <= 1 {
		panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.NewParties = n
}

func setNewT() {
	if common.TssCfg.NewThreshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := GetInt("please set new threshold(t), at least t + 1 parties needs participant signing: ", reader)
	if err != nil {
		panic(err)
	}
	if t <= 0 {
		panic(fmt.Errorf("t should greater than 0"))
	}
	if t+1 >= common.TssCfg.NewParties {
		panic(fmt.Errorf("t + 1 should less than parties"))
	}
	common.TssCfg.NewThreshold = t
}

func setOldParties(isNewParty bool) {
	reader := bufio.NewReader(os.Stdin)
	if len(common.TssCfg.Signers) >= common.TssCfg.Threshold+1 {
		if common.TssCfg.Silent {
			return
		}

		answer, err := GetString("Old monikers are already set, would you like to override them[y/N]: ", reader)
		if err != nil {
			panic(err)
		}
		if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
			common.TssCfg.Signers = common.TssCfg.Signers[:0]
		} else {
			return
		}
	}

	oldMonikers := make([]string, 0, common.TssCfg.Threshold+1)
	if isNewParty {
		peers := make([]string, 0, common.TssCfg.Parties-1)
		peer_addrs := make([]string, 0, common.TssCfg.Parties-1)
		needPeerAddr := len(common.TssCfg.BootstrapPeers) == 0 && !common.TssCfg.DefaultBootstap

		for i := 1; i <= common.TssCfg.Threshold+1; i++ {
			ithParty := humanize.Ordinal(i)
			moniker, err := GetString(fmt.Sprintf("please input moniker of the %s old party", ithParty), reader)
			if err != nil {
				panic(err)
			} else {
				oldMonikers = append(oldMonikers, moniker)
			}
			id, err := GetString(fmt.Sprintf("please input id of the %s old party", ithParty), reader)
			peers = append(peers, fmt.Sprintf("%s@%s", moniker, id))

			if needPeerAddr {
				addr, err := GetString(fmt.Sprintf("please input peer listen address of the %s old party (e.g. /ip4/127.0.0.1/tcp/27148)", ithParty), reader)
				if err != nil {
					panic(err)
				}
				peer_addrs = append(peer_addrs, addr)
			}
		}
		common.TssCfg.ExpectedPeers = peers
		common.TssCfg.PeerAddrs = peer_addrs
	} else {
		oldMonikers = append(oldMonikers, common.TssCfg.Moniker)
		for i := 1; i <= common.TssCfg.Threshold; i++ {
			ithParty := humanize.Ordinal(i)
			moniker, err := GetString(fmt.Sprintf("please input moniker of the %s old party", ithParty), reader)
			if err != nil {
				panic(err)
			} else {
				oldMonikers = append(oldMonikers, moniker)
			}
		}
	}
	common.TssCfg.Signers = oldMonikers
}

func setNewParties() {
	reader := bufio.NewReader(os.Stdin)
	if len(common.TssCfg.ExpectedNewPeers) == common.TssCfg.NewParties {
		if common.TssCfg.Silent {
			return
		}

		answer, err := GetString("New monikers are already set, would you like to override them[y/N]: ", reader)
		if err != nil {
			panic(err)
		}
		if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
			common.TssCfg.ExpectedNewPeers = common.TssCfg.ExpectedNewPeers[:0]
			common.TssCfg.NewPeerAddrs = common.TssCfg.NewPeerAddrs[:0]
		} else {
			return
		}
	}

	peers := make([]string, 0, common.TssCfg.NewParties-1)
	peer_addrs := make([]string, 0, common.TssCfg.NewParties-1)
	needPeerAddr := len(common.TssCfg.BootstrapPeers) == 0 && !common.TssCfg.DefaultBootstap
	for i := 1; i <= common.TssCfg.NewParties; i++ {
		ithParty := humanize.Ordinal(i)
		moniker, err := GetString(fmt.Sprintf("please input moniker of the %s new party", ithParty), reader)
		if err != nil {
			panic(err)
		}
		id, err := GetString(fmt.Sprintf("please input id of the %s new party", ithParty), reader)
		peers = append(peers, fmt.Sprintf("%s@%s", moniker, id))

		if needPeerAddr {
			addr, err := GetString(fmt.Sprintf("please input peer listen address of the %s new party (e.g. /ip4/127.0.0.1/tcp/27148)", ithParty), reader)
			if err != nil {
				panic(err)
			}
			peer_addrs = append(peer_addrs, addr)
		}
	}
	common.TssCfg.ExpectedNewPeers = peers
	common.TssCfg.NewPeerAddrs = peer_addrs
}
