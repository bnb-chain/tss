package p2p

import (
	"context"
	"encoding/gob"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/binance-chain/tss-lib/types"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	ifconnmgr "github.com/libp2p/go-libp2p-interface-connmgr"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/multiformats/go-multiaddr"
	"github.com/whyrusleeping/yamux"

	"github.com/binance-chain/tss/common"
)

const (
	protocalId       = "/tss/binance/0.0.1"
	loggerName       = "trans"
	receiveChBufSize = 500
)

// P2P implementation of Transporter
type p2pTransporter struct {
	ifconnmgr.NullConnMgr

	ctx context.Context

	pathToRouteTable string
	expectedPeers    []peer.ID
	streams          *sync.Map // map[peer.ID.Pretty()]inet.Stream
	bootstrapPeers   []multiaddr.Multiaddr
	relayPeers       []multiaddr.Multiaddr
	notifee          inet.Notifiee

	receiveCh chan types.Message
	host      host.Host
}

var _ ifconnmgr.ConnManager = (*p2pTransporter)(nil)
var _ common.Transporter = (*p2pTransporter)(nil)

// Constructor of p2pTransporter
// Once this is done, the transportation is ready to use
func NewP2PTransporter(config common.P2PConfig) common.Transporter {
	t := &p2pTransporter{}

	t.ctx = context.Background()
	t.pathToRouteTable = config.PathToRouteTable
	for _, expectedPeer := range config.ExpectedPeers {
		if pid, err := peer.IDB58Decode(string(expectedPeer)); err != nil {
			panic(err)
		} else {
			t.expectedPeers = append(t.expectedPeers, pid)
		}
	}
	t.streams = &sync.Map{}
	t.bootstrapPeers = config.BootstrapPeers
	// TODO: relay addr need further confirm
	// The correct address should be /p2p-circuit/p2p/<dest ID> rather than /p2p-circuit/p2p/<relay ID>
	for _, relayPeerAddr := range config.RelayPeers {
		relayPeerInfo, err := pstore.InfoFromP2pAddr(relayPeerAddr)
		if err != nil {
			panic(err)
		}
		relayAddr, err := multiaddr.NewMultiaddr("/p2p-circuit/p2p/" + relayPeerInfo.ID.Pretty())
		if err != nil {
			panic(err)
		}
		t.relayPeers = append(t.relayPeers, relayAddr)
	}
	t.notifee = &cmNotifee{t}
	t.receiveCh = make(chan types.Message, receiveChBufSize)
	// load private key of node id
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
	}

	addr, err := multiaddr.NewMultiaddr(config.ListenAddr)
	if err != nil {
		panic(err)
	}

	host, err := libp2p.New(
		t.ctx,
		libp2p.ConnectionManager(t),
		libp2p.ListenAddrs(addr),
		libp2p.Identity(privKey),
		libp2p.EnableRelay(relay.OptDiscovery),
		libp2p.NATPortMap(), // actually I cannot find a case that NATPortMap can help, but in case some edge case, created it to save relay server performance
	)
	if err != nil {
		panic(err)
	}
	host.SetStreamHandler(protocalId, t.handleStream)
	t.host = host
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	dht := t.setupDHTClient()
	t.initConnection(dht)

	return t
}

func (t *p2pTransporter) Broadcast(msg types.Message) error {
	logger.Debug("Broadcast: ", msg)
	var err error
	t.streams.Range(func(to, stream interface{}) bool {
		if e := t.Send(msg, common.TssClientId(to.(string))); e != nil {
			err = e
			return false
		}
		return true
	})
	return err
}

func (t *p2pTransporter) Send(msg types.Message, to common.TssClientId) error {
	logger.Debug("Sending: ", msg)
	// TODO: stream.Write should be protected by their lock?
	stream, ok := t.streams.Load(to.String())
	if ok && stream != nil {
		enc := gob.NewEncoder(stream.(inet.Stream))
		if err := enc.Encode(&msg); err != nil {
			return err
		}
		logger.Debug("Sent: ", msg)
	}
	return nil
}

func (t p2pTransporter) ReceiveCh() <-chan types.Message {
	return t.receiveCh
}

func (t p2pTransporter) Close() (err error) {
	logger.Info("Closing p2ptransporter")

	t.streams.Range(func(key, stream interface{}) bool {
		if stream == nil {
			return true
		}
		if e := stream.(inet.Stream).Close(); e != nil {
			err = e
			return false
		}
		return true
	})
	return nil
}

// implementation of ConnManager

func (t *p2pTransporter) Notifee() inet.Notifiee {
	return t.notifee
}

// implementation of

func (t *p2pTransporter) handleStream(stream inet.Stream) {
	pid := stream.Conn().RemotePeer().Pretty()
	logger.Infof("Connected to: %s(%s)", pid, stream.Protocol())

	t.streams.Store(pid, stream)
	// TODO: tidy this before go to production
	stream.SetDeadline(time.Now().Add(time.Hour))
	go t.readDataRoutine(stream)
}

func (t *p2pTransporter) readDataRoutine(stream inet.Stream) {
	for {
		var msg types.Message
		decoder := gob.NewDecoder(stream)
		if err := decoder.Decode(&msg); err == nil {
			t.receiveCh <- msg
		} else {
			logger.Error("failed to decode message: ", err)
			switch err {
			case yamux.ErrConnectionReset:
				break // connManager would handle the reconnection
			default:
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (t *p2pTransporter) initConnection(dht *libp2pdht.IpfsDHT) {
	wg := sync.WaitGroup{}
	for _, pid := range t.expectedPeers {
		if stream, ok := t.streams.Load(pid.Pretty()); ok && stream != nil {
			continue
		}

		if pid == t.host.ID() {
			continue
		}
		wg.Add(1)
		go t.connectRoutine(dht, pid, &wg)
	}
	wg.Wait()
}

func (t *p2pTransporter) connectRoutine(dht *libp2pdht.IpfsDHT, pid peer.ID, wg *sync.WaitGroup) {
	timeout := time.NewTimer(15 * time.Minute)
	defer func() {
		timeout.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-timeout.C:
			break
		default:
			for {
				time.Sleep(1000 * time.Millisecond)
				_, err := dht.FindPeer(t.ctx, pid)
				if err == nil {
					logger.Debug("Found peer:", pid)
				} else {
					logger.Warningf("Cannot resolve addr of peer: %s, err: %s", pid, err.Error())
					continue
				}

				logger.Debug("Connecting to:", pid)
				stream, err := t.host.NewStream(t.ctx, pid, protocol.ID(protocalId))

				if err != nil {
					logger.Info("Normal Connection failed:", err)
					if err := t.tryRelaying(pid); err != nil {
						continue
					} else {
						return
					}
				} else {
					t.handleStream(stream)
					return
				}
			}
		}
	}

}

func (t *p2pTransporter) tryRelaying(pid peer.ID) error {
	t.host.Network().(*swarm.Swarm).Backoff().Clear(pid)
	relayaddr, err := multiaddr.NewMultiaddr("/p2p-circuit/p2p/" + pid.Pretty())
	relayInfo := pstore.PeerInfo{
		ID:    pid,
		Addrs: []multiaddr.Multiaddr{relayaddr},
	}
	err = t.host.Connect(t.ctx, relayInfo)
	if err != nil {
		logger.Warning("Relay Connection failed:", err)
		return err
	}
	stream, err := t.host.NewStream(t.ctx, pid, protocalId)
	if err != nil {
		logger.Warning("Relay Stream failed:", err)
		return err
	}
	t.handleStream(stream)
	return nil
}

func (t *p2pTransporter) setupDHTClient() *libp2pdht.IpfsDHT {
	ds, err := leveldb.NewDatastore(t.pathToRouteTable, nil)
	if err != nil {
		panic(err)
	}

	kademliaDHT, err := libp2pdht.New(
		t.ctx,
		t.host,
		opts.Datastore(ds),
		opts.Client(true),
	)
	if err != nil {
		panic(err)
	}

	// Connect to bootstrap peers
	for _, bootstrapAddr := range t.bootstrapPeers {
		bootstrapPeerInfo, err := pstore.InfoFromP2pAddr(bootstrapAddr)
		if err != nil {
			panic(err)
		}
		if err := t.host.Connect(t.ctx, *bootstrapPeerInfo); err != nil {
			logger.Warning(err)
		} else {
			logger.Info("Connection established with bootstrap node:", *bootstrapPeerInfo)
		}
	}

	// Connect to relay peers to get NAT support
	// TODO: exclude relay peers that are same with bootstrap peers
	for _, relayAddr := range t.relayPeers {
		relayPeerInfo, err := pstore.InfoFromP2pAddr(relayAddr)
		if err != nil {
			panic(err)
		}
		if err := t.host.Connect(t.ctx, *relayPeerInfo); err != nil {
			logger.Warning(err)
		} else {
			logger.Info("Connection established with relay node:", *relayPeerInfo)
		}
	}

	return kademliaDHT
}
