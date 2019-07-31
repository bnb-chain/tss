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
		common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var mustBeNewParty bool
		if _, err := os.Stat(path.Join(common.TssCfg.Home, "sk.json")); os.IsNotExist(err) {
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
			bootstrap.Run(cmd, args)
			common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
			common.TssCfg.BMode = common.RegroupMode
		} else {
			setChannelId()
			setChannelPasswd()
		}

		updateConfig()

		c := client.NewTssClient(common.TssCfg, client.RegroupMode, false)
		c.Start()
	},
}

func setIsOld() {
	reader := bufio.NewReader(os.Stdin)
	answer, err := GetBool("Participant as an old committee?[Y/n]:", true, reader)
	if err != nil {
		panic(err)
	}
	if answer {
		common.TssCfg.IsOldCommittee = true
	}
}

func setIsNew() {
	reader := bufio.NewReader(os.Stdin)
	answer, err := GetBool("Participant as a new committee?[Y/n]:", true, reader)
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

func setUnknownParties() {
	reader := bufio.NewReader(os.Stdin)
	n, err := GetInt("how many peers are unknown before:", reader)
	if err != nil {
		panic(err)
	}
	common.TssCfg.UnknownParties = n
}
