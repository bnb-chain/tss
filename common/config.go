package common

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"reflect"
	"strings"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/multiformats/go-multiaddr"
)

// A new type we need for writing a custom flag parser
type addrList []multiaddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := multiaddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

type P2PConfig struct {
	ListenAddr string `mapstructure:"listen" json:"listen"`
	LogLevel   string `mapstructure:"log_level" json:"log_level"`

	// client only config
	BootstrapPeers       addrList `mapstructure:"bootstraps" json:"bootstraps"`
	RelayPeers           addrList `mapstructure:"relays" json:"relays"`
	ExpectedPeers        []string `mapstructure:"peers" json:"peers"` // expected peer list, <moniker>@<TssClientId>
	BroadcastSanityCheck bool     `mapstructure:"broadcast_sanity_check" json:"broadcast_sanity_check"`
}

type TssConfig struct {
	P2PConfig `mapstructure:"p2p" json:"p2p"`

	Id          TssClientId
	Moniker     string
	Threshold   int
	Parties     int
	Mode        string // client, server, setup
	ProfileAddr string `mapstructure:"profile_addr" json:"profile_addr"`
	Password    string
	Home        string
}

func ReadConfig() (TssConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	pflag.String("home", path.Join(home, ".tss"), "Path to config/route_table/node_key/tss_key files, configs in config file can be overriden by command line arguments")

	pflag.String("p2p.listen", "/ip4/0.0.0.0/tcp/27148", "Adds a multiaddress to the listen list")
	pflag.String("p2p.log_level", "debug", "log level")
	pflag.StringSlice("p2p.bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	pflag.StringSlice("p2p.relays", []string{}, "relay server list")
	pflag.StringSlice("p2p.peers", []string{}, "peers in this threshold scheme")
	pflag.Bool("p2p.broadcast_sanity_check", true, "whether verify broadcasted message's hash with peers")

	pflag.String("id", "", "id of current node")
	pflag.String("moniker", "", "moniker of current node")
	pflag.Int("threshold", 2, "threshold of this scheme")
	pflag.Int("parties", 3, "total parities of this scheme")
	pflag.String("mode", "client", "optional values: client,server,setup")
	pflag.String("profile_addr", "", "host:port of go pprof")
	pflag.String("password", "", "password, should only be used for testing. If empty, you will be prompted for password to save/load the secret share")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
	viper.SetConfigName("config")
	cfgPath := viper.GetString("home")
	viper.AddConfigPath(cfgPath)
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		} else {
			fmt.Println("!!!NOTICE!!! cannot find config.json, would use config in command line parameter")
		}
	}

	var config TssConfig
	err = viper.Unmarshal(&config, func(config *mapstructure.DecoderConfig) {
		config.DecodeHook = func(from, to reflect.Type, data interface{}) (interface{}, error) {
			if from.Kind() == reflect.Slice && from.Elem().Kind() == reflect.String && to == reflect.TypeOf(addrList{}) {
				var al addrList
				for _, value := range data.([]string) {
					addr, err := multiaddr.NewMultiaddr(value)
					if err != nil {
						return nil, err
					}
					al = append(al, addr)
				}
				return al, nil
			}
			if from.Kind() == reflect.Slice && from.Elem().Kind() == reflect.Interface && to == reflect.TypeOf(addrList{}) {
				var al addrList
				for _, value := range data.([]interface{}) {
					addr, err := multiaddr.NewMultiaddr(value.(string))
					if err != nil {
						return nil, err
					}
					al = append(al, addr)
				}
				return al, nil
			}
			return data, nil
		}
	})
	//err = viper.Unmarshal(&config)

	if len(config.P2PConfig.BootstrapPeers) == 0 {
		fmt.Println("!!!NOTICE!!! cannot find bootstraps servers in config, would use libp2p default bootstraps")
		config.P2PConfig.BootstrapPeers = dht.DefaultBootstrapPeers
	}
	if err != nil {
		panic(err)
	}

	if config.ProfileAddr != "" {
		go func() {
			http.ListenAndServe(config.ProfileAddr, nil)
		}()
	}

	return config, nil
}
