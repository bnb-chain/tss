package ssdp

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/koron/go-ssdp"

	"github.com/binance-chain/tss/common"
)

type SsdpService struct {
	finished      chan bool
	moniker       string
	listenAddr    string
	expectedPeers int
	monitor       *ssdp.Monitor

	PeerAddrs map[string]string // uuid -> connectable address
}

func NewSsdpService(moniker, listenAddr string, expectedPeers int) *SsdpService {
	s := &SsdpService{
		finished:      make(chan bool),
		moniker:       moniker,
		listenAddr:    listenAddr,
		expectedPeers: expectedPeers,

		PeerAddrs: make(map[string]string),
	}
	s.monitor = &ssdp.Monitor{
		Alive:  s.onAlive,
		Bye:    nil,
		Search: nil,
	}
	return s
}

func (s *SsdpService) CollectPeerAddrs() {
	ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)

	ad, err := ssdp.Advertise("my:tss", fmt.Sprintf("unique:%s", s.moniker), s.listenAddr, "", 1800)
	if err != nil {
		log.Fatal(err)
	}
	aliveTick := time.Tick(time.Duration(10) * time.Second)

	if err := s.monitor.Start(); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-aliveTick:
			ad.Alive()
		case <-s.finished:
			break
		}
	}
	ad.Bye()
	ad.Close()
	close(s.finished)
	s.monitor.Close()
}

func (s *SsdpService) stop() {
	s.finished <- true
}

func (s *SsdpService) onAlive(m *ssdp.AliveMessage) {
	if _, ok := s.PeerAddrs[m.USN]; !ok {
		multiAddrs := strings.Split(m.Location, ",")
		for _, multiAddr := range multiAddrs {
			addr, err := common.ConvertMultiAddrStrToNormalAddr(multiAddr)
			if err != nil {
				continue
			}
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				continue
			}
			err = conn.Close()
			if err != nil {
				continue
			}
			s.PeerAddrs[m.USN] = multiAddr
			break
		}
	}

	if len(s.PeerAddrs) == s.expectedPeers {
		s.stop()
	}
}
