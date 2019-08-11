package cmd

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

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
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString(flagHome), vault, passphrase); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var mustNew bool
		if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); os.IsNotExist(err) {
			mustNew = true
		}

		if !mustNew {
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

		var tssRegroup *exec.Cmd
		var tmpVault string
		if !mustNew && common.TssCfg.IsNewCommittee {
			pwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			rand.Seed(time.Now().UnixNano())
			suffix := rand.Intn(9999-1000) + 1000
			tmpVault = fmt.Sprintf("%s_%d", common.TssCfg.Vault, suffix)
			tmpMoniker := fmt.Sprintf("%s_%d", common.TssCfg.Moniker, suffix)
			devnull, err := os.Open(os.DevNull)
			if err != nil {
				panic(err)
			}

			// TODO: this relies on user doesn't rename the binary we released
			tssInit := exec.Command(path.Join(pwd, "tss"), "init", "--home", common.TssCfg.Home, "--vault_name", tmpVault, "--moniker", tmpMoniker, "--password", common.TssCfg.Password)
			tssInit.Stdin = devnull
			tssInit.Stdout = devnull

			if err := tssInit.Run(); err != nil {
				panic(fmt.Errorf("failed to fork tss init command: %v", err))
			}

			tssRegroup = exec.Command(path.Join(pwd, "tss"), "regroup", "--home", common.TssCfg.Home, "--vault_name", tmpVault, "--password", common.TssCfg.Password, "--parties", strconv.Itoa(common.TssCfg.Parties), "--threshold", strconv.Itoa(common.TssCfg.Threshold), "--new_parties", strconv.Itoa(common.TssCfg.NewParties), "--new_threshold", strconv.Itoa(common.TssCfg.NewThreshold), "--channel_password", common.TssCfg.Password, "--channel_id", common.TssCfg.ChannelId, "--log_level", common.TssCfg.LogLevel)
			stdOut, err := os.Create(path.Join(common.TssCfg.Home, tmpVault, "tss.log"))
			if err != nil {
				panic(err)
			}
			tssRegroup.Stdin = devnull
			tssRegroup.Stdout = stdOut
			tssRegroup.Stderr = stdOut

			if err := tssRegroup.Start(); err != nil {
				panic(fmt.Errorf("failed to fork tss regroup command: %v", err))
			}
		}

		common.TssCfg.BMode = common.PreRegroupMode
		bootstrapCmd.Run(cmd, args)
		common.TssCfg.BMode = common.RegroupMode

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

		if !mustNew && common.TssCfg.IsNewCommittee && tssRegroup != nil {
			err := tssRegroup.Wait()
			if err != nil {
				fmt.Errorf("failed to wait child tss process finished: %v", err)
			}

			// TODO: Make sure this works under different os (linux and windows)
			os.Rename(
				path.Join(common.TssCfg.Home, tmpVault),
				path.Join(common.TssCfg.Home, common.TssCfg.Vault))
		}
	},
}

func setIsOld() {
	reader := bufio.NewReader(os.Stdin)
	answer, err := common.GetBool("Participant as an old (signing) committee?[Y/n]:", true, reader)
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
