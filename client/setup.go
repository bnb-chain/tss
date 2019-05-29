package client

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/binance-chain/tss/common"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"os"
)

func Setup(cfg common.TssConfig) {
	err := os.Mkdir("./configs", 0700)
	if err != nil {
		panic(err)
	}
	allPeerIds := make([]common.TssClientId, 0, cfg.Parties)
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
		allPeerIds = append(allPeerIds, common.TssClientId(pid.Pretty()))

		bytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(configPath+"/node_key", bytes, os.FileMode(0600))
	}

	for i := 0; i < cfg.Parties; i++ {
		configFilePath := fmt.Sprintf("./configs/%d/config.json", i)
		tssConfig := cfg
		tssConfig.P2PConfig.ExpectedPeers = make([]common.TssClientId, len(allPeerIds), len(allPeerIds))
		copy(tssConfig.P2PConfig.ExpectedPeers, allPeerIds)
		tssConfig.P2PConfig.ExpectedPeers = append(tssConfig.P2PConfig.ExpectedPeers[:i], tssConfig.P2PConfig.ExpectedPeers[i+1:]...)

		tssConfig.Id = allPeerIds[i]
		tssConfig.Moniker = fmt.Sprintf("party%d", i)
		tssConfig.Mode = "client"

		bytes, err := json.MarshalIndent(&tssConfig, "", "    ")
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(configFilePath, bytes, os.FileMode(0600))
	}
}
