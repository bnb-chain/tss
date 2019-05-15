package common

import (
	"flag"
	dht "github.com/libp2p/go-libp2p-kad-dht"
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
	PathToNodeKey    string
	PathToRouteTable string
	ListenAddr       string
	LogLevel         string

	// client only config
	BootstrapPeers addrList
	RelayPeers     addrList
	ExpectedPeers  cidList
}

type TssConfig struct {
	P2PConfig

	Id           TssClientId
	Threshold    int
	TotalClients int
	Mode         string
}

func ParseFlags() (TssConfig, error) {
	config := TssConfig{}

	flag.StringVar(&config.PathToNodeKey, "node_key", "./node_key", "Path to node key")
	flag.StringVar(&config.PathToRouteTable, "route_table", "./rt", "Path to DHT route table store")
	flag.StringVar(&config.ListenAddr, "listen", "/ip4/0.0.0.0/tcp/27148", "Adds a multiaddress to the listen list")
	flag.StringVar(&config.LogLevel, "log_level", "debug", "log level")
	flag.Var(&config.BootstrapPeers, "bootstraps", "bootstrap server list")
	flag.Var(&config.RelayPeers, "relays", "relay server list")
	flag.Var(&config.ExpectedPeers, "peers", "peers in this threshold scheme")

	flag.Var(&config.Id, "id", "id of current node")
	flag.IntVar(&config.Threshold, "threshold", 2, "threshold of this scheme")
	flag.IntVar(&config.TotalClients, "TotalClients", 3, "total nodes of this scheme")
	flag.StringVar(&config.Mode, "mode", "client", "client or server")

	flag.Parse()

	if len(config.BootstrapPeers) == 0 {
		config.BootstrapPeers = dht.DefaultBootstrapPeers
	}

	return config, nil
}
