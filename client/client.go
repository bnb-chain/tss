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

	"github.com/bgentry/speakeasy"
	lib "github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/ecdsa/regroup"
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

type ClientMode uint8

const (
	KeygenMode ClientMode = iota
	SignMode
	RegroupMode
)

func (m ClientMode) String() string {
	switch m {
	case KeygenMode:
		return "keygen"
	case SignMode:
		return "sign"
	case RegroupMode:
		return "regroup"
	default:
		panic("unknown mode")
	}
}

type TssClient struct {
	config      common.TssConfig
	localParty  tss.Party
	transporter common.Transporter

	params        *tss.Parameters
	regroupParams *tss.ReGroupParameters
	key           *keygen.LocalPartySaveData
	signature     []byte

	saveCh chan keygen.LocalPartySaveData
	signCh chan signing.LocalPartySignData
	sendCh chan tss.Message

	mode ClientMode
}

func init() {
	gob.RegisterName("LocalPartySaveData", keygen.LocalPartySaveData{})
	gob.Register(p2p.P2pMessageWithHash{})
}

func NewTssClient(config common.TssConfig, mode ClientMode, mock bool) *TssClient {
	id := string(config.Id)
	key := lib.SHA512_256([]byte(id)) // TODO: discuss should we really need pass p2p nodeid pubkey into NewPartyID? (what if in memory implementation)
	partyID := tss.NewPartyID(id, config.Moniker, new(big.Int).SetBytes(key))
	unsortedPartyIds := make(tss.UnSortedPartyIDs, 0, config.Parties)
	if mode == RegroupMode {
		if common.TssCfg.IsOldCommittee {
			unsortedPartyIds = append(unsortedPartyIds, partyID)
		}
	} else {
		unsortedPartyIds = append(unsortedPartyIds, partyID)
	}
	unsortedNewPartyIds := make(tss.UnSortedPartyIDs, 0, config.NewParties)

	signers := make(map[string]int, 0) // used by sign and regroup mode for filtering correct shares from LocalPartySaveData
	if mode == SignMode || mode == RegroupMode {
		if mode == SignMode {
			common.TssCfg.BMode = common.SignMode
		}
		if mode == RegroupMode {
			common.TssCfg.BMode = common.RegroupMode
		}
		bootstrapper := &common.Bootstrapper{
			ChannelId:       config.ChannelId,
			ChannelPassword: config.ChannelPassword,
			Cfg:             &common.TssCfg,
		}
		t := p2p.NewP2PTransporter(config.Home, config.Id.String(), bootstrapper, &config.P2PConfig)
		t.Shutdown()
		bootstrapper.Peers.Range(func(_, value interface{}) bool {
			if pi, ok := value.(common.PeerInfo); ok {
				if mode == SignMode || (mode == RegroupMode && pi.IsOld) {
					config.Signers = append(config.Signers, pi.Moniker)
				}
			}
			return true
		})
		if mode == SignMode || (mode == RegroupMode && common.TssCfg.IsOldCommittee) {
			config.Signers = append(config.Signers, config.Moniker)
		}

		if len(config.Signers) < config.Threshold+1 {
			panic(fmt.Errorf("no enough signers (%d) to meet requirement: %d", len(config.Signers), config.Threshold+1))
		}
		updatePeerOriginalIndexes(config, partyID, signers)

		if mode == RegroupMode {
			bootstrapper.Peers.Range(func(_, value interface{}) bool {
				if pi, ok := value.(common.PeerInfo); ok {
					if pi.IsNew {
						config.ExpectedNewPeers = append(config.ExpectedNewPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
						config.NewPeerAddrs = append(config.NewPeerAddrs, pi.RemoteAddr)
					}
				}
				return true
			})
		}
	}

	signingExpectedPeers := make([]string, 0, config.Parties) // used to override peers
	signingExpectedAddrs := make([]string, 0, config.Parties)
	if !mock {
		for i, peer := range config.P2PConfig.ExpectedPeers {
			id := string(p2p.GetClientIdFromExpecetdPeers(peer))
			moniker := p2p.GetMonikerFromExpectedPeers(peer)
			key := lib.SHA512_256([]byte(id))
			if mode == SignMode || mode == RegroupMode {
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
		if mode == RegroupMode {
			for _, peer := range config.P2PConfig.ExpectedNewPeers {
				id := string(p2p.GetClientIdFromExpecetdPeers(peer))
				moniker := p2p.GetMonikerFromExpectedPeers(peer)
				key := lib.SHA512_256([]byte(id))
				if moniker != config.Moniker {
					unsortedNewPartyIds = append(unsortedNewPartyIds,
						tss.NewPartyID(
							id,
							moniker,
							new(big.Int).SetBytes(key)))
				}
			}
			if config.IsNewCommittee {
				unsortedNewPartyIds = append(unsortedNewPartyIds, partyID)
			}
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

		mode: mode,
	}
	var localParty tss.Party
	if mode == KeygenMode {
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		localParty = keygen.NewLocalParty(params, sendCh, saveCh)
		c.localParty = localParty
		logger.Infof("[%s] initialized localParty: %s", config.Moniker, localParty)
	} else if mode == SignMode {
		key := loadSavedKeyForSign(config, sortedIds, signers)
		pubKey := btcec.PublicKey(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()})
		logger.Infof("[%s] public key: %X\n", config.Moniker, pubKey.SerializeCompressed())
		address, _ := getAddress(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()})
		logger.Infof("[%s] address is: %s\n", config.Moniker, address)
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		c.key = &key
		c.params = params
	} else if mode == RegroupMode {
		sortedNewIds := tss.SortPartyIDs(unsortedNewPartyIds)
		newP2pCtx := tss.NewPeerContext(sortedNewIds)
		params := tss.NewReGroupParameters(
			p2pCtx,
			newP2pCtx,
			partyID,
			config.Parties,
			config.Threshold,
			config.NewParties,
			config.NewThreshold)
		c.regroupParams = params

		if _, ok := signers[common.TssCfg.Moniker]; ok {
			key := loadSavedKeyForRegroup(config, sortedIds, signers)
			c.key = &key
			localParty = regroup.NewLocalParty(params, key, sendCh, saveCh)
		} else {
			// TODO do this better!
			save := keygen.LocalPartySaveData{
				BigXj:       make([]*crypto.ECPoint, common.TssCfg.NewParties),
				PaillierPks: make([]*paillier.PublicKey, common.TssCfg.NewParties),
				NTildej:     make([]*big.Int, common.TssCfg.NewParties),
				H1j:         make([]*big.Int, common.TssCfg.NewParties),
				H2j:         make([]*big.Int, common.TssCfg.NewParties),
			}
			localParty = regroup.NewLocalParty(params, save, sendCh, saveCh)
		}
		c.localParty = localParty
	}

	if mock {
		c.transporter = p2p.GetMemTransporter(config.Id)
	} else {
		if mode == SignMode {
			config.ExpectedPeers = signingExpectedPeers
			config.PeerAddrs = signingExpectedAddrs
		}
		if mode == RegroupMode {
			config.ExpectedPeers = signingExpectedPeers
			config.PeerAddrs = signingExpectedAddrs

			for i, newPeer := range config.ExpectedNewPeers {
				exist := false
				for _, existPeer := range config.ExpectedPeers {
					if newPeer == existPeer {
						exist = true
						break
					}
				}
				if !exist {
					config.ExpectedPeers = append(config.ExpectedPeers, config.ExpectedNewPeers[i])
					config.PeerAddrs = append(config.PeerAddrs, config.NewPeerAddrs[i])
				}
			}
		}
		// will block until peers are connected
		c.transporter = p2p.NewP2PTransporter(config.Home, config.Id.String(), nil, &config.P2PConfig)
	}

	return &c
}

func (client *TssClient) Start() {
	switch client.mode {
	case KeygenMode:
		if err := client.localParty.Start(); err != nil {
			panic(err)
		}

		done := make(chan bool)
		go client.sendMessageRoutine(client.sendCh)
		go client.saveDataRoutine(client.saveCh, done)
		//go c.sendDummyMessageRoutine()
		go client.handleMessageRoutine()
		<-done
	case RegroupMode:
		if err := client.localParty.Start(); err != nil {
			panic(err)
		}

		done := make(chan bool)
		go client.sendMessageRoutine(client.sendCh)
		go client.saveDataRoutine(client.saveCh, done)
		//go c.sendDummyMessageRoutine()
		go client.handleMessageRoutine()
		<-done
	case SignMode:
		message, ok := big.NewInt(0).SetString(client.config.Message, 10)
		if !ok {
			panic(fmt.Errorf("message to be sign: %s is not a valid big.Int", client.config.Message))
		}
		client.signImpl(message)
	}
}

func (client *TssClient) handleMessageRoutine() {
	for msg := range client.transporter.ReceiveCh() {
		logger.Infof("[%s] received message: %s", client.config.Moniker, msg)
		ok, err := client.localParty.Update(msg, client.mode.String())
		if err != nil {
			logger.Errorf("[%s] error updating local party state: %v", client.config.Moniker, err)
		} else if !ok {
			logger.Warningf("[%s] Update still waiting for round to finish", client.config.Moniker)
		} else {
			logger.Debugf("[%s] update success", client.config.Moniker)
		}
	}
}

func (client *TssClient) sendMessageRoutine(sendCh <-chan tss.Message) {
	for msg := range sendCh {
		dest := msg.GetTo()
		if dest == nil || len(dest) > 1 {
			client.transporter.Broadcast(msg)
		} else {
			client.transporter.Send(msg, common.TssClientId(dest[0].ID))
		}
	}
}

func (client *TssClient) saveDataRoutine(saveCh <-chan keygen.LocalPartySaveData, done chan<- bool) {
	for msg := range saveCh {
		// Used for debugging signature verification failed issue, never uncomment in production!
		//plainJson, err := json.Marshal(msg)
		//ioutil.WriteFile(path.Join(client.config.Home, "plain.json"), plainJson, 0400)

		if client.mode == RegroupMode {
			isNewCommitee := false
			for _, peer := range common.TssCfg.ExpectedNewPeers {
				id := p2p.GetClientIdFromExpecetdPeers(peer)
				if id == common.TssCfg.Id {
					isNewCommitee = true
					break
				}
			}
			if !isNewCommitee {
				continue
			}
		}

		logger.Infof("[%s] received save data", client.config.Moniker)
		address, err := getAddress(ecdsa.PublicKey{tss.EC(), msg.ECDSAPub.X(), msg.ECDSAPub.Y()})
		if err != nil {
			logger.Errorf("[%s] failed to generate address from public key :%v", client.config.Moniker, err)
		} else {
			logger.Infof("[%s] bech32 address is: %s", client.config.Moniker, address)
		}

		wPriv, err := os.OpenFile(path.Join(client.config.Home, "sk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		defer wPriv.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		wPub, err := os.OpenFile(path.Join(client.config.Home, "pk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		defer wPub.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		Save(&msg, client.transporter.NodeKey(), client.config.KDFConfig, client.config.Password, wPriv, wPub)

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

func loadSavedKeyForSign(config common.TssConfig, sortedIds tss.SortedPartyIDs, signers map[string]int) keygen.LocalPartySaveData {
	result := loadSavedKey(config)
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
		filteredPaillierPks,
		filteredNTildej,
		filteredH1j,
		filteredH2j,
		result.Index,
		filteredKs,
		result.ECDSAPub,
	}

	return filteredResult
}

func loadSavedKeyForRegroup(config common.TssConfig, sortedIds tss.SortedPartyIDs, signers map[string]int) keygen.LocalPartySaveData {
	result := loadSavedKeyForSign(config, sortedIds, signers)

	if config.IsNewCommittee {
		// TODO: negociate with Luke to see how to fill non-loaded keys here
		for i := len(signers); i < config.NewParties; i++ {
			result.BigXj = append(result.BigXj, result.BigXj[len(signers)-1])
			result.PaillierPks = append(result.PaillierPks, result.PaillierPks[len(signers)-1])
			result.NTildej = append(result.NTildej, result.NTildej[len(signers)-1])
			result.H1j = append(result.H1j, result.H1j[len(signers)-1])
			result.H2j = append(result.H2j, result.H2j[len(signers)-1])
			result.Ks = append(result.Ks, result.Ks[len(signers)-1])
		}
	}
	return result
}

func loadSavedKey(config common.TssConfig) keygen.LocalPartySaveData {
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
	var passphrase string
	if config.Password != "" {
		passphrase = config.Password
	} else {
		if p, err := speakeasy.Ask("please input password to secure secret key:"); err == nil {
			passphrase = p
		} else {
			panic(err)
		}
	}

	result, _ := Load(passphrase, wPriv, wPub) // TODO: validate nodeKey
	return *result
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

// assign original keygen index to signers (old parties in regroup)
func updatePeerOriginalIndexes(config common.TssConfig, partyID *tss.PartyID, signers map[string]int) {
	for _, signer := range config.Signers {
		signers[signer] = 0
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
