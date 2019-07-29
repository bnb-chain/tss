package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "create home directory of a new tss setup, generate p2p key pair",
	Long:  "",
	PreRun: func(cmd *cobra.Command, args []string) {
		home := viper.GetString("home")
		makeHomeDir(home)
		common.ReadConfigFromHome(viper.GetViper(), home)
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		setMoniker()
		setP2pKey()
		setListenAddr()
		generateConfigFile()

		fmt.Printf("Local party has been initialized under: %s\n", common.TssCfg.Home)
		fmt.Printf("Please share following information to your peers:\n")
		fmt.Printf("************************************************************\n")
		fmt.Printf("moniker: %s\nid: %s\nlisten: %s\n", common.TssCfg.Moniker, common.TssCfg.Id, common.TssCfg.ListenAddr)
		fmt.Printf("************************************************************\n")
	},
}

func makeHomeDir(home string) {
	if _, err := os.Stat(home); err == nil {
		// home already exists
		reader := bufio.NewReader(os.Stdin)
		answer, err := GetString("Home already exist, do you like override it[y/N]: ", reader)
		if err != nil {
			panic(err)
		}
		if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
			if err := os.Remove(path.Join(home, "config.json")); err != nil {
				panic(err)
			}
			if err := os.Remove(path.Join(home, "node_key")); err != nil {
				panic(err)
			}
		} else {
			return
		}
	} else {
		if err := os.Mkdir(home, 0700); err != nil {
			panic(err)
		}
	}
}

func setMoniker() {
	if common.TssCfg.Moniker != "" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	moniker, err := GetString("please set moniker of this party: ", reader)
	if err != nil {
		panic(err)
	}
	if strings.Contains(moniker, "@") {
		panic(fmt.Errorf("moniker should not contains @ sign"))
	}
	common.TssCfg.Moniker = moniker
}

func setP2pKey() {
	privKey, id, err := p2p.NewP2pPrivKey()
	if err != nil {
		panic(err)
	}

	bytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(path.Join(common.TssCfg.Home, "node_key"), bytes, os.FileMode(0600)); err != nil {
		panic(err)
	}

	common.TssCfg.Id = common.TssClientId(id.String())
}

func setListenAddr() {
	if common.TssCfg.ListenAddr != "" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	addr, err := GetString("please set listen multiaddr of this party: (/ip4/0.0.0.0/tcp/27148)", reader)
	if err != nil {
		panic(err)
	}
	if addr == "" {
		addr = "/ip4/0.0.0.0/tcp/27148"
	}
	common.TssCfg.ListenAddr = addr
}

func generateConfigFile() {
	bytes, err := json.MarshalIndent(&common.TssCfg, "", "    ")
	if err != nil {
		panic(err)
	}
	if err = ioutil.WriteFile(path.Join(common.TssCfg.Home, "config.json"), bytes, os.FileMode(0600)); err != nil {
		panic(err)
	}
}
