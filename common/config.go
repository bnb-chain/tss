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
	PeerAddrs            []string `mapstructure:"peer_addrs" json:"peer_addrs"` // used for some peer has known connectable ip:port so that connection to them doesn't require bootstrap and relay nodes. i.e. in a LAN environment, if ip ports are preallocated, BootstrapPeers and RelayPeers can be empty with all parties host port set
	ExpectedPeers        []string `mapstructure:"peers" json:"peers"`           // expected peer list, <moniker>@<TssClientId>
	DefaultBootstap      bool     `mapstructure:"default_bootstrap", json:"default_bootstrap"`
	BroadcastSanityCheck bool     `mapstructure:"broadcast_sanity_check" json:"broadcast_sanity_check"`
}

// Argon2 parameters, setting should refer 9th section of https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
type KDFConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 `mapstructure:"salt_length" json:"salt_length"`
	KeyLength   uint32 `mapstructure:"key_length" json:"key_length"`
}

func DefaultKDFConfig() KDFConfig {
	return KDFConfig{
		65536,
		13,
		4,
		16,
		48,
	}
}

type TssConfig struct {
	P2PConfig `mapstructure:"p2p" json:"p2p"`
	KDFConfig `mapstructure:"kdf" json:"kdf"`

	Id          TssClientId
	Moniker     string
	Threshold   int
	Parties     int
	Mode        string // keygen, sign, server, setup
	ProfileAddr string `mapstructure:"profile_addr" json:"profile_addr"`
	Password    string
	Message     string   // string represented big.Int, will refactor later
	Signers     []string // monikers of signers
	Indexes     []string // indexes of signers,
	// has to be string here as viper's intSlice support seems doesn't work:
	// https://github.com/spf13/viper/issues/613, TODO: generate this automatically
	Home string
}

func bindP2pConfigs() {
	pflag.String("p2p.listen", "/ip4/0.0.0.0/tcp/27148", "Adds a multiaddress to the listen list")
	pflag.String("p2p.log_level", "debug", "log level")
	pflag.StringSlice("p2p.bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	pflag.StringSlice("p2p.relays", []string{}, "relay server list")
	pflag.StringSlice("p2p.peemor_addrs", []string{}, "peer's multiple addresses")
	pflag.StringSlice("p2p.peers", []string{}, "peers in this threshold scheme")
	pflag.Bool("p2p.default_bootstrap", false, "whether to use default bootstrap")
	pflag.Bool("p2p.broadcast_sanity_check", true, "whether verify broadcasted message's hash with peers")
}

// more detail explaination of these parameters can be found:
// https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
// https://www.alexedwards.net/blog/how-to-hash-and-verify-passwords-with-argon2-in-go
func bindKdfConfigs() {
	pflag.Uint32("kdf.memory", 65536, "The amount of memory used by the algorithm (in kibibytes)")
	pflag.Uint32("kdf.iterations", 13, "The number of iterations (or passes) over the memory.")
	pflag.Uint8("kdf.parallelism", 4, "The number of threads (or lanes) used by the algorithm.")
	pflag.Uint32("kdf.salt_length", 16, "Length of the random salt. 16 bytes is recommended for password hashing.")
	pflag.Uint32("kdf.key_length", 48, "Length of the generated key (or password hash). must be 32 bytes or more")
}

func bindClientConfigs() {
	pflag.String("id", "", "id of current node")
	pflag.String("moniker", "", "moniker of current node")
	pflag.Int("threshold", 1, "threshold of this scheme")
	pflag.Int("parties", 3, "total parities of this scheme")
	pflag.String("mode", "keygen", "optional values: keygen,sign,server,setup")
	pflag.String("profile_addr", "", "host:port of go pprof")
	pflag.String("password", "", "password, should only be used for testing. If empty, you will be prompted for password to save/load the secret share")
	pflag.String("message", "", "message(in *big.Int.String() format) to be signed, only used in sign mode")
	pflag.StringSlice("signers", []string{}, "monikers of singers separated by comma")
	pflag.StringSlice("indexes", []string{}, "indexes of signers during keygen phase")
}

func ReadConfig() (TssConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	pflag.String("home", path.Join(home, ".tss"), "Path to config/route_table/node_key/tss_key files, configs in config file can be overriden by command line arguments")

	bindP2pConfigs()
	bindKdfConfigs()
	bindClientConfigs()

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
	return ReadConfigFromHome(viper.GetViper(), viper.GetString("home"))
}

func ReadConfigFromHome(v *viper.Viper, home string) (TssConfig, error) {
	v.SetConfigName("config")
	v.AddConfigPath(home)
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		} else {
			fmt.Println("!!!NOTICE!!! cannot find config.json, would use config in command line parameter")
		}
	}

	var config TssConfig
	err = v.Unmarshal(&config, func(config *mapstructure.DecoderConfig) {
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
	if err != nil {
		panic(err)
	}

	// validate configs
	if len(config.P2PConfig.BootstrapPeers) == 0 {
		fmt.Println("!!!NOTICE!!! cannot find bootstraps servers in config")
		if config.P2PConfig.DefaultBootstap {
			fmt.Println("!!!NOTICE!!! Would use libp2p's default bootstraps")
			config.P2PConfig.BootstrapPeers = dht.DefaultBootstrapPeers
		}
	}
	if config.KDFConfig.KeyLength < 32 {
		panic("Length of the generated key (or password hash). must be 32 bytes or more")
	}

	if config.ProfileAddr != "" {
		go func() {
			http.ListenAndServe(config.ProfileAddr, nil)
		}()
	}

	return config, nil
}
