package cmd

import (
	"bufio"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign a transaction",
	Long:  "sign a transaction using local share, signers will be prompted to fill in",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		setChannelId()
		setChannelPasswd()
		setMessage()

		c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
		c.Start()
	},
}

// TODO: use MessageBridge
func setMessage() {
	if message := viper.GetString("message"); message != "" {
		common.TssCfg.Message = message
		return
	}

	reader := bufio.NewReader(os.Stdin)
	message, err := common.GetString("please set message(in *big.Int.String() format) to be signed: ", reader)
	if err != nil {
		common.Panic(err)
	}
	common.TssCfg.Message = message
}
