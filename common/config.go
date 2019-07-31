package common

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"strings"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/multiformats/go-multiaddr"
)

var TssCfg TssConfig

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
	NewPeerAddrs         []string `mapstructure:"new_peer_addrs" json:"-"`      // same with `PeerAddrs` but for new parties for regroup
	ExpectedNewPeers     []string `mapstructure:"new_peers" json:"-"`           // expected new peer list used for regroup, <moniker>@<TssClientId>, after regroup success, this field will replace ExpectedPeers
	DefaultBootstap      bool     `mapstructure:"default_bootstrap", json:"default_bootstrap"`
	BroadcastSanityCheck bool     `mapstructure:"broadcast_sanity_check" json:"broadcast_sanity_check"`
}

func DefaultP2PConfig() P2PConfig {
	return P2PConfig{
		LogLevel:             "INFO",
		BroadcastSanityCheck: true,
	}
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

	Id      TssClientId
	Moniker string

	Threshold    int
	Parties      int
	NewThreshold int `mapstructure:"new_threshold" json:"new_threshold"`
	NewParties   int `mapstructure:"new_parties" json:"new_parties"`

	ProfileAddr     string   `mapstructure:"profile_addr" json:"profile_addr"`
	Password        string   `json:"-"`
	Message         string   `json:"-"` // string represented big.Int, will refactor later
	Signers         []string `json:"-"` // monikers of signers for signing transaction or regroup, self moniker should be included
	ChannelId       string   `mapstructure:"channel_id" json:"-"`
	ChannelPassword string   `mapstructure:"channel_password" json:"-"`

	IsOldCommittee bool          `mapstructure:"is_old" json:"-"`
	IsNewCommittee bool          `mapstructure:"is_new" json:"-"`
	UnknownParties int           `json:"-"`
	BMode          BootstrapMode `json:"-"`

	Silent bool
	Home   string
}

func DefaultTssConfig() TssConfig {
	return TssConfig{
		P2PConfig: DefaultP2PConfig(),
		KDFConfig: DefaultKDFConfig(),
	}
}

func ReadConfigFromHome(v *viper.Viper, home string) error {
	v.SetConfigName("config")
	v.AddConfigPath(home)
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
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
		return err
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
		return fmt.Errorf("Length of the generated key (or password hash). must be 32 bytes or more")
	}

	if config.ProfileAddr != "" {
		go func() {
			http.ListenAndServe(config.ProfileAddr, nil)
		}()
	}

	TssCfg = config
	return nil
}
