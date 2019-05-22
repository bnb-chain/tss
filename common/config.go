package common

import (
	"flag"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"reflect"
	"strings"

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

type cidList []TssClientId

func (cl *cidList) String() string {
	strs := make([]string, len(*cl))
	for i, cid := range *cl {
		strs[i] = string(cid)
	}
	return strings.Join(strs, ",")
}

func (cl *cidList) Set(value string) error {
	*cl = append(*cl, TssClientId(value))
	return nil
}

type P2PConfig struct {
	PathToNodeKey    string `mapstructure:"node_key"`
	PathToRouteTable string `mapstructure:"route_table"`
	ListenAddr       string `mapstructure:"listen"`
	LogLevel         string `mapstructure:"log_level"`

	// client only config
	BootstrapPeers addrList `mapstructure:"bootstraps"`
	RelayPeers     addrList `mapstructure:"relays"`
	ExpectedPeers  cidList  `mapstructure:"peers"`
}

type TssConfig struct {
	P2PConfig `mapstructure:",squash"`

	Id        TssClientId
	Moniker   string
	Index     int
	Threshold int
	Parties   int
	Mode      string // client, server, setup
}

func ReadConfig() (TssConfig, error) {
	pflag.String("node_key", "./node_key", "Path to node key")
	pflag.String("route_table", "./rt", "Path to DHT route table store")
	pflag.String("listen", "/ip4/0.0.0.0/tcp/27148", "Adds a multiaddress to the listen list")
	pflag.String("log_level", "debug", "log level")
	pflag.StringSlice("bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	pflag.StringSlice("relays", []string{}, "relay server list")
	pflag.StringSlice("peers", []string{}, "peers in this threshold scheme")

	pflag.String("id", "", "id of current node")
	pflag.String("moniker", "", "moniker of current node")
	pflag.Int("threshold", 2, "threshold of this scheme")
	pflag.Int("parties", 3, "total parities of this scheme")
	pflag.String("mode", "client", "client,server,setup")

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		panic(err)
	} else {
		fmt.Printf("!!!NOTICE!!! cannot find config.json, would use config in command line parameter")
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

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
			return data, nil
		}
	})

	if len(config.BootstrapPeers) == 0 {
		fmt.Printf("!!!NOTICE!!! cannot find bootstraps servers in config, would use libp2p default bootstraps")
		config.BootstrapPeers = dht.DefaultBootstrapPeers
	}
	if err != nil {
		panic(err)
	}

	return config, nil
}
