package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"

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
		passphrase := askPassphrase()
		vault := askVault()
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString(flagHome), vault, passphrase); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var mustBeNewParty bool
		if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); os.IsNotExist(err) {
			mustBeNewParty = true
		}

		if !mustBeNewParty {
			askPassphrase()
			setIsOld()
			setIsNew()
		} else {
			common.TssCfg.IsOldCommittee = false
			common.TssCfg.IsNewCommittee = true
			setPassphrase()
			setOldN()
			setOldT()
		}
		setNewN()
		setNewT()
		setUnknownParties()
		if common.TssCfg.UnknownParties > 0 {
			common.TssCfg.BMode = common.PreRegroupMode
			bootstrapCmd.Run(cmd, args)
			common.TssCfg.BMode = common.RegroupMode
		} else {
			setChannelId()
			setChannelPasswd()
		}

		c := client.NewTssClient(&common.TssCfg, client.RegroupMode, false)
		c.Start()

		if common.TssCfg.IsNewCommittee {
			common.TssCfg.ExpectedPeers = common.TssCfg.ExpectedNewPeers
			common.TssCfg.PeerAddrs = common.TssCfg.NewPeerAddrs
			common.TssCfg.ExpectedNewPeers = common.TssCfg.ExpectedNewPeers[:]
			common.TssCfg.NewPeerAddrs = common.TssCfg.NewPeerAddrs[:]
			common.TssCfg.Parties = common.TssCfg.NewParties
			common.TssCfg.Threshold = common.TssCfg.NewThreshold
			common.TssCfg.NewParties = 0
			common.TssCfg.NewThreshold = 0
			updateConfig()
		}
	},
}

func setIsOld() {
	reader := bufio.NewReader(os.Stdin)
	answer, err := common.GetBool("Participant as an old committee?[Y/n]:", true, reader)
	if err != nil {
		panic(err)
	}
	if answer {
		common.TssCfg.IsOldCommittee = true
	}
}

func setIsNew() {
	reader := bufio.NewReader(os.Stdin)
	answer, err := common.GetBool("Participant as a new committee?[Y/n]:", true, reader)
	if err != nil {
		panic(err)
	}
	if answer {
		common.TssCfg.IsNewCommittee = true
	}
}

func setOldN() {
	if common.TssCfg.Parties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := common.GetInt("please set old total parties(n) (default: 3): ", 3, reader)
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
	t, err := common.GetInt("please set old threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
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
	n, err := common.GetInt("please set new total parties(n) (default 3): ", 3, reader)
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
	t, err := common.GetInt("please set new threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
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

func setUnknownParties() {
	if common.TssCfg.UnknownParties != -1 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := common.GetInt("how many peers are unknown before (default 0): ", 0, reader)
	if err != nil {
		panic(err)
	}
	common.TssCfg.UnknownParties = n
}
