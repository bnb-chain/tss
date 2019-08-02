package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/bgentry/speakeasy"
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
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString("home")); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		checkBootstrap(cmd, args)
		checkN()
		setT()
		setPassphrase()
		updateConfig()

		c := client.NewTssClient(&common.TssCfg, client.KeygenMode, false)
		c.Start()
	},
}

func checkBootstrap(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	if len(common.TssCfg.ExpectedPeers) > 0 {
		answer, err := GetBool("Do you like re-bootstrap again?[y/N]: ", false, reader)
		if err != nil {
			panic(err)
		}
		if answer {
			bootstrapCmd.Run(cmd, args)
		}
	} else {
		bootstrapCmd.Run(cmd, args)
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
	t, err := GetInt("please set threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
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

func setPassphrase() {
	if common.TssCfg.Password != "" {
		return
	}

	if p, err := speakeasy.Ask("please set password to secure secret key:"); err == nil {
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
