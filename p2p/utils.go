package p2p

import (
	"fmt"
	"github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	"strings"
	"time"
)

var logger = log.Logger(loggerName)

func DumpDHTRoutine(dht *libp2pdht.IpfsDHT) {
	for {
		dht.RoutingTable().Print()
		time.Sleep(10 * time.Second)
	}
}

func DumpPeersRoutine(host host.Host) {
	for {
		time.Sleep(10 * time.Second)
		builder := strings.Builder{}
		for _, peer := range host.Network().Peers() {
			fmt.Fprintf(&builder, "%s\n", peer)
		}
		logger.Debugf("Dump peers:\n%s", builder.String())
	}
}
