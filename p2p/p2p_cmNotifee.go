package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/multiformats/go-multiaddr"
)

type cmNotifee struct {
	t *p2pTransporter
}

var _ inet.Notifiee = (*cmNotifee)(nil)

func (nn *cmNotifee) Connected(n inet.Network, c inet.Conn) {
	logger.Debugf("[Connected] %s (%s)", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String())
}

func (nn *cmNotifee) Disconnected(n inet.Network, c inet.Conn) {
	logger.Debugf("[Disconnected] %s (%s)", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String())
	//nn.t.streams.Delete(c.RemotePeer().Pretty())
	// TODO: trigger reconnect
}

func (nn *cmNotifee) Listen(n inet.Network, addr multiaddr.Multiaddr) {}

func (nn *cmNotifee) ListenClose(n inet.Network, addr multiaddr.Multiaddr) {}

func (nn *cmNotifee) OpenedStream(n inet.Network, s inet.Stream) {
	logger.Debugf("[OpenedStream] %s (%s)", s.Conn().RemotePeer().Pretty(), s.Protocol())
}

func (nn *cmNotifee) ClosedStream(n inet.Network, s inet.Stream) {
	logger.Debugf("[ClosedStream] %s (%s)", s.Conn().RemotePeer().Pretty(), s.Protocol())
}
