package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
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
		passphrase := setPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), home, passphrase); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		setP2pKey()
		setListenAddr()
		setMoniker()
		updateConfig()

		addr, err := multiaddr.NewMultiaddr(common.TssCfg.ListenAddr)
		if err != nil {
			panic(err)
		}
		host, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addr))
		if err != nil {
			panic(err)
		}
		client.Logger.Debugf("this node will be listen on %s", host.Addrs())
		err = host.Close()
		if err != nil {
			panic(err)
		}
		client.Logger.Infof("Local party has been initialized under: %s\n", common.TssCfg.Home)
	},
}

func makeHomeDir(home string) {
	if _, err := os.Stat(home); err == nil {
		// home already exists
		reader := bufio.NewReader(os.Stdin)
		answer, err := GetBool("Home already exist, do you like override it[y/N]: ", false, reader)
		if err != nil {
			panic(err)
		}
		if answer {
			if _, err := os.Stat(path.Join(home, "config.json")); err == nil {
				if err := os.Remove(path.Join(home, "config.json")); err != nil {
					panic(err)
				}
			}
			if _, err := os.Stat(path.Join(home, "node_key")); err == nil {
				if err := os.Remove(path.Join(home, "node_key")); err != nil {
					panic(err)
				}
			}
			if _, err := os.Stat(path.Join(home, "pk.json")); err == nil {
				if err := os.Remove(path.Join(home, "pk.json")); err != nil {
					panic(err)
				}
			}
			if _, err := os.Stat(path.Join(home, "sk.json")); err == nil {
				if err := os.Remove(path.Join(home, "sk.json")); err != nil {
					panic(err)
				}
			}
		} else {
			fmt.Println("nothing happened")
			os.Exit(0)
		}
	} else {
		if err := os.Mkdir(home, 0700); err != nil {
			panic(err)
		}
	}
}

func setPassphrase() string {
	if pw := viper.GetString("password"); pw != "" {
		return pw
	}

	if p, err := speakeasy.Ask("please set password to secure secret key:"); err == nil {
		if p2, err := speakeasy.Ask("please input again:"); err == nil {
			if p2 != p {
				panic(fmt.Errorf("two inputs does not match, please start again"))
			} else {
				checkComplexityOfPassword(p)
				viper.Set("password", p)
				return p
			}
		} else {
			panic(err)
		}
	} else {
		panic(err)
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

	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}
	common.TssCfg.ListenAddr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)
}

func updateConfig() {
	err := common.SaveConfig(&common.TssCfg)
	if err != nil {
		panic(err)
	}
}
