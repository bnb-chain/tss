package client

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math/big"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/bgentry/speakeasy"
	lib "github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/bech32"
	"github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

var logger = log.Logger("tss")

type TssClient struct {
	config      common.TssConfig
	localParty  tss.Party
	transporter common.Transporter

	params    *tss.Parameters
	key       *keygen.LocalPartySaveData
	signature []byte

	saveCh chan keygen.LocalPartySaveData
	signCh chan signing.LocalPartySignData
	sendCh chan tss.Message
}

func init() {
	gob.RegisterName("LocalPartySaveData", keygen.LocalPartySaveData{})
	gob.Register(p2p.P2pMessageWithHash{})
}

func NewTssClient(config common.TssConfig, mock bool) *TssClient {

	id := string(config.Id)
	key := lib.SHA512_256([]byte(id)) // TODO: discuss should we really need pass p2p nodeid pubkey into NewPartyID? (what if in memory implementation)
	partyID := tss.NewPartyID(id, config.Moniker, new(big.Int).SetBytes(key))
	unsortedPartyIds := make(tss.UnSortedPartyIDs, 0, config.Parties)
	unsortedPartyIds = append(unsortedPartyIds, partyID)

	signers := make(map[string]int, 0) // used for filtering correct shares from LocalPartySaveData
	if config.Mode == "sign" {
		if len(config.Signers) < config.Threshold+1 {
			panic(fmt.Errorf("no enough signers (%d) to meet requirement: %d", len(config.Signers), config.Threshold+1))
		}

		for _, signer := range config.Signers {
			signers[signer] = 0
		}

		if _, ok := signers[config.Moniker]; !ok {
			panic(fmt.Errorf("this node is not in signers list"))
		}

		allPartyIds := make(tss.UnSortedPartyIDs, 0, config.Parties) // all parties, used for calculating party's index during keygen
		allPartyIds = append(allPartyIds, partyID)
		for _, peer := range config.P2PConfig.ExpectedPeers {
			id := string(p2p.GetClientIdFromExpecetdPeers(peer))
			moniker := p2p.GetMonikerFromExpectedPeers(peer)
			key := lib.SHA512_256([]byte(id))
			allPartyIds = append(allPartyIds,
				tss.NewPartyID(
					id,
					moniker,
					new(big.Int).SetBytes(key)))
		}

		sortedIds := tss.SortPartyIDs(allPartyIds)
		for _, id := range sortedIds {
			if _, ok := signers[id.Moniker]; ok {
				signers[id.Moniker] = id.Index
			}
		}
	}

	signingExpectedPeers := make([]string, 0, config.Parties) // used to override peers
	signingExpectedAddrs := make([]string, 0, config.Parties)
	if !mock {
		for i, peer := range config.P2PConfig.ExpectedPeers {
			id := string(p2p.GetClientIdFromExpecetdPeers(peer))
			moniker := p2p.GetMonikerFromExpectedPeers(peer)
			key := lib.SHA512_256([]byte(id))
			if config.Mode == "sign" {
				if _, ok := signers[moniker]; !ok {
					continue
				}
				signingExpectedPeers = append(signingExpectedPeers, peer)
				if len(config.P2PConfig.PeerAddrs) == len(config.P2PConfig.ExpectedPeers) {
					signingExpectedAddrs = append(signingExpectedAddrs, config.P2PConfig.PeerAddrs[i])
				}
			}
			unsortedPartyIds = append(unsortedPartyIds,
				tss.NewPartyID(
					id,
					moniker,
					new(big.Int).SetBytes(key)))
		}
	} else {
		for i := 0; i < config.Parties; i++ {
			id, _ := strconv.Atoi(string(config.Id))
			if i != id {
				id := strconv.Itoa(i)
				key := lib.SHA512_256([]byte(id))
				unsortedPartyIds = append(unsortedPartyIds, tss.NewPartyID(id, id, new(big.Int).SetBytes(key)))
			}
		}
	}
	sortedIds := tss.SortPartyIDs(unsortedPartyIds)
	p2pCtx := tss.NewPeerContext(sortedIds)
	saveCh := make(chan keygen.LocalPartySaveData)
	signCh := make(chan signing.LocalPartySignData)
	sendCh := make(chan tss.Message, len(sortedIds)*10*2) // max signing messages 10 times hash confirmation messages
	c := TssClient{
		config: config,

		saveCh: saveCh,
		signCh: signCh,
		sendCh: sendCh,
	}
	var localParty tss.Party
	if config.Mode == "keygen" {
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		localParty = keygen.NewLocalParty(params, sendCh, saveCh)
		c.localParty = localParty
		logger.Infof("[%s] initialized localParty: %s", config.Moniker, localParty)
	} else if config.Mode == "sign" {
		key := LoadSavedKey(config, sortedIds, signers)
		pubKey := btcec.PublicKey(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()})
		logger.Infof("[%s] public key: %X\n", config.Moniker, pubKey.SerializeCompressed())
		address, _ := getAddress(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()})
		logger.Infof("[%s] address is: %s\n", config.Moniker, address)
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		c.key = &key
		c.params = params
	}

	if mock {
		c.transporter = p2p.GetMemTransporter(config.Id)
	} else {
		if config.Mode == "sign" {
			config.ExpectedPeers = signingExpectedPeers
			config.PeerAddrs = signingExpectedAddrs
		}
		// will block until peers are connected
		c.transporter = p2p.NewP2PTransporter(config.Home, config.P2PConfig)
	}

	return &c
}

func (client *TssClient) Start() {
	if client.config.Mode == "sign" {
		message, ok := big.NewInt(0).SetString(client.config.Message, 10)
		if !ok {
			panic(fmt.Errorf("message to be sign: %s is not a valid big.Int", client.config.Message))
		}
		client.signImpl(message)
	} else {
		if err := client.localParty.Start(); err != nil {
			panic(err)
		}

		done := make(chan bool)
		go client.sendMessageRoutine(client.sendCh)
		go client.saveDataRoutine(client.saveCh, done)
		//go c.sendDummyMessageRoutine()
		go client.handleMessageRoutine()
		<-done
	}
}

func (client *TssClient) handleMessageRoutine() {
	for msg := range client.transporter.ReceiveCh() {
		logger.Infof("[%s] received message: %s", client.config.Moniker, msg)
		ok, err := client.localParty.Update(msg, client.config.Mode)
		if err != nil {
			logger.Errorf("[%s] error updating local party state: %v", client.config.Moniker, err)
		} else if !ok {
			logger.Warningf("[%s] Update still waiting for round to finish", client.config.Moniker)
		} else {
			logger.Debugf("[%s] update success", client.config.Moniker)
		}
	}
}

// just used for debugging p2p communication
func (client *TssClient) sendDummyMessageRoutine() {
	for {
		client.transporter.Broadcast(common.DummyMsg{string(client.config.Id)})
		time.Sleep(10 * time.Second)
	}
}

func (client *TssClient) sendMessageRoutine(sendCh <-chan tss.Message) {
	for msg := range sendCh {
		dest := msg.GetTo()
		if dest == nil {
			client.transporter.Broadcast(msg)
		} else {
			client.transporter.Send(msg, common.TssClientId(dest.ID))
		}
	}
}

func (client *TssClient) saveDataRoutine(saveCh <-chan keygen.LocalPartySaveData, done chan<- bool) {
	for msg := range saveCh {
		// Used for debugging signature verification failed issue, never uncomment in production!
		//plainJson, err := json.Marshal(msg)
		//ioutil.WriteFile(path.Join(client.config.Home, "plain.json"), plainJson, 0400)

		logger.Infof("[%s] received save data", client.config.Moniker)
		address, err := getAddress(ecdsa.PublicKey{tss.EC(), msg.ECDSAPub.X(), msg.ECDSAPub.Y()})
		if err != nil {
			logger.Errorf("[%s] failed to generate address from public key :%v", client.config.Moniker, err)
		} else {
			logger.Infof("[%s] bech32 address is: %s", client.config.Moniker, address)
		}

		var passphrase string
		if client.config.Password != "" {
			passphrase = client.config.Password
		} else {
			if p, err := speakeasy.Ask("please input password to secure secret key:"); err == nil {
				passphrase = p
			} else {
				panic(err)
			}
		}

		wPriv, err := os.OpenFile(path.Join(client.config.Home, "sk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			panic(err)
		}
		defer wPriv.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		wPub, err := os.OpenFile(path.Join(client.config.Home, "pk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			panic(err)
		}
		defer wPub.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		Save(&msg, client.transporter.NodeKey(), client.config.KDFConfig, passphrase, wPriv, wPub)

		if done != nil {
			done <- true
			close(done)
		}
		break
	}
}

func (client *TssClient) saveSignatureRoutine(signCh <-chan signing.LocalPartySignData, done chan<- bool) {
	for signature := range signCh {
		client.signature = signature.Signature
		if done != nil {
			done <- true
			close(done)
		}
		break
	}
}

func LoadSavedKey(config common.TssConfig, sortedIds tss.SortedPartyIDs, signers map[string]int) keygen.LocalPartySaveData {
	wPriv, err := os.OpenFile(path.Join(config.Home, "sk.json"), os.O_RDONLY, 0400)
	if err != nil {
		panic(err)
	}
	defer wPriv.Close()
	wPub, err := os.OpenFile(path.Join(config.Home, "pk.json"), os.O_RDONLY, 0400)
	if err != nil {
		panic(err)
	}
	defer wPub.Close()
	result, _ := Load(config.Password, wPriv, wPub) // TODO: validate nodeKey
	filteredBigXj := make([]*crypto.ECPoint, 0)
	filteredPaillierPks := make([]*paillier.PublicKey, 0)
	filteredNTildej := make([]*big.Int, 0)
	filteredH1j := make([]*big.Int, 0)
	filteredH2j := make([]*big.Int, 0)
	filteredKs := make([]*big.Int, 0)
	for _, partyId := range sortedIds {
		keygenIdx := signers[partyId.Moniker]
		filteredBigXj = append(filteredBigXj, result.BigXj[keygenIdx])
		filteredPaillierPks = append(filteredPaillierPks, result.PaillierPks[keygenIdx])
		filteredNTildej = append(filteredNTildej, result.NTildej[keygenIdx])
		filteredH1j = append(filteredH1j, result.H1j[keygenIdx])
		filteredH2j = append(filteredH2j, result.H2j[keygenIdx])
		filteredKs = append(filteredKs, result.Ks[keygenIdx])
	}
	filteredResult := keygen.LocalPartySaveData{
		result.Xi,
		result.ShareID,
		result.PaillierSk,
		filteredBigXj,
		result.ECDSAPub,
		filteredPaillierPks,
		filteredNTildej,
		filteredH1j,
		filteredH2j,
		result.Index,
		filteredKs,
	}

	return filteredResult
}

func getAddress(key ecdsa.PublicKey) (string, error) {
	btcecPubKey := btcec.PublicKey(key)
	// be consistent with tendermint/crypto
	compressed := btcecPubKey.SerializeCompressed()
	hasherSHA256 := sha256.New()
	hasherSHA256.Write(compressed[:]) // does not error
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error

	address := []byte(hasherRIPEMD160.Sum(nil))
	converted, err := bech32.ConvertBits(address, 8, 5, true) // TODO: error check
	if err != nil {
		return "", errors.Wrap(err, "encoding bech32 failed")
	}
	return bech32.Encode("tbnb", converted)
}
