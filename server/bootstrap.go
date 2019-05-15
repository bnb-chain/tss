package server

import (
	"context"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
	"github.com/ipfs/go-ds-leveldb"
	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"os"
)

var logger = log.Logger("srv")

type TssBootstrapServer struct{}

func NewTssBootstrapServer(config common.P2PConfig) *TssBootstrapServer {
	bs := TssBootstrapServer{}

	var privKey crypto.PrivKey
	if _, err := os.Stat(config.PathToNodeKey); err == nil {
		bytes, err := ioutil.ReadFile(config.PathToNodeKey)
		if err != nil {
			panic(err)
		}
		privKey, err = crypto.UnmarshalPrivateKey(bytes)
		if err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}

	addr, err := multiaddr.NewMultiaddr(config.ListenAddr)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(addr),
		libp2p.Identity(privKey),
		libp2p.EnableRelay(relay.OptHop),
		libp2p.NATPortMap(),
	)
	if err != nil {
		panic(err)
	}

	ds, err := leveldb.NewDatastore(config.PathToRouteTable, nil)
	if err != nil {
		panic(err)
	}

	kademliaDHT, err := libp2pdht.New(
		ctx,
		host,
		opts.Datastore(ds),
		opts.Client(false))
	if err != nil {
		panic(err)
	}

	go p2p.DumpDHTRoutine(kademliaDHT)
	go p2p.DumpPeersRoutine(host)

	return &bs
}
