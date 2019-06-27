package client

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

func Setup(cfg common.TssConfig) {
	err := os.Mkdir("./configs", 0700)
	if err != nil {
		panic(err)
	}
	allPeerIds := make([]string, 0, cfg.Parties)
	for i := 0; i < cfg.Parties; i++ {
		configPath := fmt.Sprintf("./configs/%d", i)
		err := os.Mkdir(configPath, 0700)
		if err != nil {
			panic(err)
		}
		// generate node identifier key
		privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			panic(err)
		}

		pid, err := peer.IDFromPublicKey(privKey.GetPublic())
		if err != nil {
			panic(err)
		}
		allPeerIds = append(allPeerIds, fmt.Sprintf("%s@%s", fmt.Sprintf("party%d", i), pid.Pretty()))

		bytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(configPath+"/node_key", bytes, os.FileMode(0600))
	}

	for i := 0; i < cfg.Parties; i++ {
		configFilePath := fmt.Sprintf("./configs/%d/config.json", i)
		tssConfig := cfg
		tssConfig.P2PConfig.ExpectedPeers = make([]string, cfg.Parties, cfg.Parties)
		copy(tssConfig.P2PConfig.ExpectedPeers, allPeerIds)
		tssConfig.P2PConfig.ExpectedPeers = append(tssConfig.P2PConfig.ExpectedPeers[:i], tssConfig.P2PConfig.ExpectedPeers[i+1:]...)

		if cfg.Parties == len(cfg.P2PConfig.PeerAddrs) {
			tssConfig.P2PConfig.PeerAddrs = make([]string, cfg.Parties, cfg.Parties)
			copy(tssConfig.P2PConfig.PeerAddrs, cfg.P2PConfig.PeerAddrs)
			tssConfig.P2PConfig.PeerAddrs = append(tssConfig.P2PConfig.PeerAddrs[:i], tssConfig.P2PConfig.PeerAddrs[i+1:]...)

			//for idx, peer := range tssConfig.P2PConfig.ExpectedPeers {
			//	tssConfig.P2PConfig.PeerAddrs[idx] += "/p2p/" + string(p2p.GetClientIdFromExpecetdPeers(peer))
			//}
		}

		tssConfig.Id = p2p.GetClientIdFromExpecetdPeers(allPeerIds[i])
		tssConfig.Moniker = p2p.GetMonikerFromExpectedPeers(allPeerIds[i])
		tssConfig.Mode = "keygen"

		bytes, err := json.MarshalIndent(&tssConfig, "", "    ")
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(configFilePath, bytes, os.FileMode(0600))
	}
}
