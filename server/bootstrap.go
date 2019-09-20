package server

import (
	"context"
	"io/ioutil"
	"os"
	"path"

	"github.com/ipfs/go-ds-leveldb"
	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/multiformats/go-multiaddr"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

var logger = log.Logger("srv")

type TssP2PServer struct{}

func NewTssP2PServer(home, vault string, config common.P2PConfig) *TssP2PServer {
	bs := TssP2PServer{}

	var privKey crypto.PrivKey
	pathToNodeKey := path.Join(home, vault, "node_key")
	if _, err := os.Stat(pathToNodeKey); err == nil {
		bytes, err := ioutil.ReadFile(pathToNodeKey)
		if err != nil {
			common.Panic(err)
		}
		privKey, err = crypto.UnmarshalPrivateKey(bytes)
		if err != nil {
			common.Panic(err)
		}
	} else {
		common.Panic(err)
	}

	addr, err := multiaddr.NewMultiaddr(config.ListenAddr)
	if err != nil {
		common.Panic(err)
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
		common.Panic(err)
	}

	ds, err := leveldb.NewDatastore(path.Join(home, vault, "rt/"), nil)
	if err != nil {
		common.Panic(err)
	}

	kademliaDHT, err := libp2pdht.New(
		ctx,
		host,
		opts.Datastore(ds),
		opts.Client(false))
	if err != nil {
		common.Panic(err)
	}

	go p2p.DumpDHTRoutine(kademliaDHT)
	go p2p.DumpPeersRoutine(host)

	logger.Infof("Bootstrap server has started: %v, %s", host.Addrs(), host.ID().Pretty())

	return &bs
}
