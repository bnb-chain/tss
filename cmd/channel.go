package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/cobra"

	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(channel)
}

var channel = &cobra.Command{
	Use:              "channel",
	Short:            "generate a channel id for bootstrapping",
	TraverseChildren: false,
	Run: func(cmd *cobra.Command, args []string) {
		channelId, err := rand.Int(rand.Reader, big.NewInt(999))
		if err != nil {
			panic(err)
		}
		expireTime := time.Now().Add(30 * time.Minute).Unix()
		fmt.Printf("channel id: %s\n", fmt.Sprintf("%.3d%s", channelId.Int64(), common.ConvertTimestampToHex(expireTime)))
	},
}
