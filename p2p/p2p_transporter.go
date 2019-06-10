package p2p

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/binance-chain/tss-lib/types"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-host"
	ifconnmgr "github.com/libp2p/go-libp2p-interface-connmgr"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
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
	encoders         map[common.TssClientId]*gob.Encoder
	numOfStreams     int32 // atomic int of len(streams)
	bootstrapPeers   []multiaddr.Multiaddr
	relayPeers       []multiaddr.Multiaddr
	notifee          inet.Notifiee

	// sanity check related field
	broadcastSanityCheck bool
	sanityCheckMtx       *sync.Mutex
	ioMtx                *sync.Mutex
	pendingCheckHashMsg  map[p2pMessageKey]*P2pMessageWithHash   // guarded by sanityCheckMtx
	receivedPeersHashMsg map[p2pMessageKey][]*P2pMessageWithHash // guarded by sanityCheckMtx

	receiveCh chan types.Message
	host      host.Host
}

// encapsulation of messages that need to be broadcasted
// only send/receive this message on broadcast_sanity_check turn on
type P2pMessageWithHash struct {
	types.MessageMetadata
	Hash      [sha256.Size]byte
	originMsg *types.Message
}

func (m P2pMessageWithHash) GetType() string {
	return fmt.Sprintf("[Hash]%s", m.MessageMetadata.GetType())
}

func (m P2pMessageWithHash) String() string {
	return fmt.Sprintf("[Hash]%s, hash:%x", m.MessageMetadata.String(), m.Hash)
}

type p2pMessageKey string

func keyOf(m P2pMessageWithHash) p2pMessageKey {
	return p2pMessageKey(fmt.Sprintf("%s%s", m.GetFrom().ID, m.MessageMetadata.GetType()))
}

var _ ifconnmgr.ConnManager = (*p2pTransporter)(nil)
var _ common.Transporter = (*p2pTransporter)(nil)

// Constructor of p2pTransporter
// Once this is done, the transportation is ready to use
func NewP2PTransporter(home string, config common.P2PConfig) common.Transporter {
	t := &p2pTransporter{}

	t.ctx = context.Background()
	t.pathToRouteTable = path.Join(home, "rt/")
	for _, expectedPeer := range config.ExpectedPeers {
		if pid, err := peer.IDB58Decode(string(GetClientIdFromExpecetdPeers(expectedPeer))); err != nil {
			panic(err)
		} else {
			t.expectedPeers = append(t.expectedPeers, pid)
		}
	}
	t.streams = &sync.Map{}
	t.encoders = make(map[common.TssClientId]*gob.Encoder)
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
	t.broadcastSanityCheck = config.BroadcastSanityCheck
	if t.broadcastSanityCheck {
		t.sanityCheckMtx = &sync.Mutex{}
		t.pendingCheckHashMsg = make(map[p2pMessageKey]*P2pMessageWithHash)
		t.receivedPeersHashMsg = make(map[p2pMessageKey][]*P2pMessageWithHash)
	}
	t.ioMtx = &sync.Mutex{}

	t.receiveCh = make(chan types.Message, receiveChBufSize)
	// load private key of node id
	var privKey crypto.PrivKey
	pathToNodeKey := path.Join(home, "node_key")
	if _, err := os.Stat(pathToNodeKey); err == nil {
		bytes, err := ioutil.ReadFile(pathToNodeKey)
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
	t.ioMtx.Lock()
	defer t.ioMtx.Unlock()

	logger.Debugf("Sending: %s, To: %s", msg, to)
	// TODO: stream.Write should be protected by their lock?
	stream, ok := t.streams.Load(to.String())
	if ok && stream != nil {
		enc := t.encoders[to]
		if err := enc.Encode(&msg); err != nil {
			return err
		}
		logger.Debugf("Sent: %s, To: %s", msg, to)
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

	if _, loaded := t.streams.LoadOrStore(pid, stream); !loaded {
		t.encoders[common.TssClientId(pid)] = gob.NewEncoder(stream)
		atomic.AddInt32(&t.numOfStreams, 1)
	}
}

func (t *p2pTransporter) readDataRoutine(pid string, stream inet.Stream) {
	decoder := gob.NewDecoder(stream)
	for {
		var msg types.Message
		if err := decoder.Decode(&msg); err == nil {
			logger.Debugf("Received message: %s from peer: %s", msg.String(), pid)

			switch m := msg.(type) {
			case P2pMessageWithHash:
				if t.broadcastSanityCheck {
					key := keyOf(m)
					t.sanityCheckMtx.Lock()
					t.receivedPeersHashMsg[key] = append(t.receivedPeersHashMsg[key], &m)
					if t.verifiedPeersBroadcastMsgGuarded(key) {
						t.receiveCh <- *t.pendingCheckHashMsg[key].originMsg
						delete(t.pendingCheckHashMsg, key)
					}
					t.sanityCheckMtx.Unlock()
				} else {
					logger.Errorf("peer %s configuration is not consistent - sanity check is enabled", pid)
				}
			case types.Message:
				if t.broadcastSanityCheck && m.GetTo() == nil {
					// we cannot use gob encoding here because the type spec registered relies on message sequence
					// in other word, it might be not deterministic https://stackoverflow.com/a/33228913/1147187
					if jsonBytes, err := json.Marshal(msg); err == nil {
						hash := sha256.Sum256(jsonBytes)
						logger.Debugf("Encoded message %s: %x (hash: %x)", m, jsonBytes, hash)
						msgWithHash := P2pMessageWithHash{types.MessageMetadata{m.GetTo(), m.GetFrom(), m.GetType()}, hash, &msg}
						t.sanityCheckMtx.Lock()
						t.pendingCheckHashMsg[keyOf(msgWithHash)] = &msgWithHash
						for _, p := range t.expectedPeers {
							if p.Pretty() != m.GetFrom().ID {
								// send our hashing of this message
								var msgToSend types.Message
								msgToSend = msgWithHash
								t.Send(msgToSend, common.TssClientId(p.Pretty()))
							}
						}
						if t.verifiedPeersBroadcastMsgGuarded(keyOf(msgWithHash)) {
							t.receiveCh <- m
							delete(t.pendingCheckHashMsg, keyOf(msgWithHash))
						}
						t.sanityCheckMtx.Unlock()
					} else {
						panic(fmt.Errorf("failed to marshal message: %s to json: %v", msg, err))
					}
				} else {
					t.receiveCh <- msg
				}
			}
		} else {
			logger.Error("failed to decode message: ", err)
			switch err {
			case yamux.ErrConnectionReset:
				break // connManager would handle the reconnection
			default:
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// guarded by t.sanityCheckMtx
func (t *p2pTransporter) verifiedPeersBroadcastMsgGuarded(key p2pMessageKey) bool {
	if t.pendingCheckHashMsg[key] == nil {
		logger.Debugf("didn't receive the main message: %s yet", key)
		return false
	} else if len(t.receivedPeersHashMsg[key])+1 != len(t.expectedPeers) {
		logger.Debugf("didn't receive enough peer's hash messages: %s yet", key)
		return false
	} else {
		for _, hashMsg := range t.receivedPeersHashMsg[key] {
			if hashMsg.Hash != t.pendingCheckHashMsg[key].Hash {
				panic("someone in network is malicious") // TODO: better logging, i.e. log which one is malicious in what way
			}
		}

		delete(t.receivedPeersHashMsg, key)
		return true
	}
}

func (t *p2pTransporter) initConnection(dht *libp2pdht.IpfsDHT) {
	for _, pid := range t.expectedPeers {
		if stream, ok := t.streams.Load(pid.Pretty()); ok && stream != nil {
			continue
		}

		if pid == t.host.ID() {
			continue
		}
		go t.connectRoutine(dht, pid)
	}

	for atomic.LoadInt32(&t.numOfStreams) < int32(len(t.expectedPeers)) {
		time.Sleep(10 * time.Millisecond)
	}
	t.streams.Range(func(pid, stream interface{}) bool {
		go t.readDataRoutine(pid.(string), stream.(inet.Stream))
		return true
	})
}

func (t *p2pTransporter) connectRoutine(dht *libp2pdht.IpfsDHT, pid peer.ID) {
	timeout := time.NewTimer(15 * time.Minute)
	defer func() {
		timeout.Stop()
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

				if atomic.LoadInt32(&t.numOfStreams) == int32(len(t.expectedPeers)) {
					return
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
