package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/multiformats/go-multiaddr"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
)

const protocalId = "/tss/binance/0.0.1"

var logger = log.Logger("tss")

func dumpDHTRoutine(dht *libp2pdht.IpfsDHT) {
	for {
		time.Sleep(10 * time.Second)
		//dht.RoutingTable().Print()
	}
}

func dumpPeersRoutine(host host.Host) {
	for {
		time.Sleep(10 * time.Second)
		builder := strings.Builder{}
		for _, peer := range host.Network().Peers() {
			fmt.Fprintf(&builder, "%s\n", peer)
		}
		logger.Debugf("Dump peers:\n%s", builder.String())
	}
}

func main() {
	log.SetLogLevel("tss", "debug")
	dhtServerMode := flag.Bool("dht_sever_mode", false, "true - start in dht_server mode")
	bootstrapPeer := flag.String("dht_server_addr", "/ip4/0.0.0.0/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1", "address of bootstrap server")
	pathToNodeKey := flag.String("node_key", "node_key", "specify a path to node_key")
	// change this for client
	listenAddr := flag.String("listen_addr", "/ip4/0.0.0.0/tcp/27148", "address this node should listen on")
	flag.Parse()

	ctx := context.Background()

	var privKey crypto.PrivKey
	if _, err := os.Stat(*pathToNodeKey); err == nil {
		bytes, err := ioutil.ReadFile(*pathToNodeKey)
		if err != nil {
			panic(err)
		}
		privKey, err = crypto.UnmarshalPrivateKey(bytes)
		if err != nil {
			panic(err)
		}
	} else if os.IsNotExist(err) {
		privKey, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			panic(err)
		}
		bytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(*pathToNodeKey, bytes, os.FileMode(0600))
	} else {
		panic(err)
	}
	addr, _ := multiaddr.NewMultiaddr(*listenAddr)
	var relayOpt libp2p.Option
	if *dhtServerMode {
		relayOpt = libp2p.EnableRelay(relay.OptHop)
	} else {
		relayOpt = libp2p.EnableRelay(relay.OptDiscovery)
	}
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(addr),
		libp2p.Identity(privKey),
		libp2p.NATPortMap(), // actually I cannot find a case that NATPortMap can help, but in case some edge case, created it to save relay server performance
		relayOpt,
	)
	if err != nil {
		panic(err)
	}
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	host.SetStreamHandler(protocalId, handleStream)

	kademliaDHT, err := libp2pdht.New(
		ctx,
		host,
		opts.Datastore(dssync.MutexWrap(ds.NewMapDatastore())),
		opts.Client(!*dhtServerMode))
	if err != nil {
		panic(err)
	}

	if *dhtServerMode {
		go dumpDHTRoutine(kademliaDHT)
	}
	go dumpPeersRoutine(host)

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	if !*dhtServerMode {
		bootstrapAddr, _ := multiaddr.NewMultiaddr(*bootstrapPeer)
		bootstrapPeerInfo, err := peerstore.InfoFromP2pAddr(bootstrapAddr)
		if err != nil {
			panic(err)
		}
		if err := host.Connect(ctx, *bootstrapPeerInfo); err != nil {
			logger.Warning(err)
		} else {
			logger.Info("Connection established with bootstrap node:", *bootstrapPeerInfo)
		}

		// We use a rendezvous point "meet me here" to announce our location.
		// This is like telling your friends to meet you at the Eiffel Tower.
		logger.Info("Announcing ourselves...")
		routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
		discovery.Advertise(ctx, routingDiscovery, "test")
		logger.Debug("Successfully announced!")

		// Now, look for others who have announced
		// This is like your friend telling you the location to meet you.
		logger.Debug("Searching for other peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, "test")
		if err != nil {
			panic(err)
		}

		for peer := range peerChan {
			if peer.ID == host.ID() {
				continue
			}
			logger.Debugf("Found peer: %s", peer)

			logger.Debug("Connecting to:", peer)
			stream, err := host.NewStream(ctx, peer.ID, protocol.ID(protocalId))

			if err != nil {
				logger.Info("Normal Connection failed:", err)
				// fallback to relaying
				host.Network().(*swarm.Swarm).Backoff().Clear(peer.ID)
				relayaddr, err := multiaddr.NewMultiaddr("/p2p-circuit/p2p/" + bootstrapPeerInfo.ID.Pretty())
				if err != nil {
					panic(err)
				}
				relayInfo := peerstore.PeerInfo{
					ID:    peer.ID,
					Addrs: []multiaddr.Multiaddr{relayaddr},
				}
				err = host.Connect(ctx, relayInfo)
				if err != nil {
					logger.Warning("Relay Connection failed:", err)
					continue
				}
				stream, err := host.NewStream(ctx, peer.ID, protocalId)
				if err != nil {
					logger.Warning("Relay Stream failed:", err)
					continue
				}
				handleStream(stream)
			} else {
				handleStream(stream)
			}

			logger.Info("Connected to:", peer)
		}
	}

	select {}
}

func handleStream(stream inet.Stream) {
	logger.Info("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(stream.Conn().LocalPeer(), rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(id peer.ID, rw *bufio.ReadWriter) {
	for {
		var err error

		_, err = rw.WriteString(fmt.Sprintf("[%s]%s\n", time.Now().String(), id.String()))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}

		time.Sleep(10 * time.Second)
	}
}
