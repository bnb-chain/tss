package blockchainUtils

import (
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Safulet/chain-integration/v2/blockchain"
	"github.com/Safulet/chain-integration/v2/blockchain/binance"
	"github.com/Safulet/chain-integration/v2/blockchain/cosmos"
	"github.com/Safulet/chain-integration/v2/blockchain/eos"
	"github.com/Safulet/chain-integration/v2/blockchain/ethereum"
	"github.com/Safulet/chain-integration/v2/blockchain/icx"
	"github.com/Safulet/chain-integration/v2/blockchain/neo"
	"github.com/Safulet/chain-integration/v2/blockchain/ontology"
	"github.com/Safulet/chain-integration/v2/blockchain/ripple"
	"github.com/Safulet/chain-integration/v2/blockchain/solana"
	"github.com/Safulet/chain-integration/v2/blockchain/stellar"
	"github.com/Safulet/chain-integration/v2/blockchain/tezos"
	"github.com/Safulet/chain-integration/v2/blockchain/tron"
	"github.com/Safulet/chain-integration/v2/blockchain/utxo_coin"
	"github.com/Safulet/chain-integration/v2/blockchain/vechain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/decred/dcrd/dcrec/edwards/v2"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func GetChainId(network string, useTestnet bool) int64 {
	switch network {
	case "ETH":
		if useTestnet {
			return ChainId[Ropsten]
		} else {
			return ChainId[Ethereum]
		}
	case "BSC":
		if useTestnet {
			return ChainId[BscTestnet]
		} else {
			return ChainId[Bsc]
		}
	case "POLYGON":
		if useTestnet {
			return ChainId[Mumbai]
		} else {
			return ChainId[Polygon]
		}
	}
	return 0
}

func InitBlockchain(network, fromAddress string, addrPubkeyHex string, useTestnet bool) (blockchain.Blockchain, error) {
	pubkeySlice, err := hex.DecodeString(addrPubkeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key on signRequest, original address message might be invalid")
	}
	var pubkey secp256k1.PubKeySecp256k1
	copy(pubkey[:], pubkeySlice)

	switch network {
	case "BNB":
		if useTestnet {
			return &binance.Binance{PublicKey: pubkey, Network: binance.Testnet}, nil
		} else {
			return &binance.Binance{PublicKey: pubkey, Network: binance.Mainnet}, nil
		}
	case "ETH":
		if useTestnet {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Ropsten], ChainId[Ropsten]), nil
		} else {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Ethereum], ChainId[Ethereum]), nil
		}
	case "BSC":
		if useTestnet {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[BscTestnet], ChainId[BscTestnet]), nil
		} else {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Bsc], ChainId[Bsc]), nil
		}
	case "XRP":
		if useTestnet {
			return &ripple.Ripple{PublicKey: pubkey, Network: ripple.Testnet}, nil
		} else {
			return &ripple.Ripple{PublicKey: pubkey, Network: ripple.Mainnet}, nil
		}
	case "BTC":
		var coinType utxo_coin.CoinType
		if useTestnet {
			coinType = utxo_coin.BITCOIN_TESTNET
		} else {
			coinType = utxo_coin.BITCOIN
		}
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[coinType],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
			SegWit:         false,
		}, nil
	case "BTCSEGWIT":
		var coinType utxo_coin.CoinType
		if useTestnet {
			coinType = utxo_coin.BITCOIN_TESTNET
		} else {
			coinType = utxo_coin.BITCOIN
		}
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[coinType],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
			SegWit:         true,
		}, nil
	case "BCH":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		networkSwitch := utxo_coin.BITCOINCASH
		if useTestnet {
			networkSwitch = utxo_coin.BITCOINCASH_TESTNET
		}
		utxo := &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[networkSwitch],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
		}
		if _, err = utxo.GetAddress(btcecPubKey.SerializeCompressed()); err != nil {
			return nil, err
		}
		return utxo, nil
	case "ATOM":
		return &cosmos.Cosmos{
			PublicKey: pubkey,
			Network:   cosmos.CosmosMainnet,
		}, nil
	case "DASH":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[utxo_coin.DASH],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
		}, nil
	case "DOGE":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[utxo_coin.DOGECOIN],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
		}, nil
	case "ETC":
		return &ethereum.EthereumClassic{Network: ethereum.MainnetClassic}, nil
	case "ICX":
		//btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		//if err != nil {
		//	return nil, err
		//}
		//pubkeyBytes := make([]byte, btcec.PubKeyBytesLenUncompressed, btcec.PubKeyBytesLenUncompressed)
		//copy(pubkeyBytes[:], btcecPubKey.SerializeUncompressed())
		chain := &icx.Icon{}
		//addrFromServer, _ := chain.GetAddress(pubkeyBytes)
		return chain, nil
	case "LTC":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		var coinType utxo_coin.CoinType
		if useTestnet {
			coinType = utxo_coin.LITECOIN_TESTNET
		} else {
			coinType = utxo_coin.LITECOIN
		}
		utxo := &utxo_coin.UtxoCoin{
			CoinConfig: utxo_coin.CoinConfigs[coinType],
		}
		if _, err = utxo.GetAddress(btcecPubKey.SerializeCompressed()); err != nil {
			return nil, err
		}
		return utxo, nil
	case "ONT":
		return &ontology.Ontology{
			PublicKey: pubkeySlice,
			Network:   ontology.Mainnet,
		}, nil
	case "QTUM":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[utxo_coin.QTUM],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
		}, nil
	case "RVN":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		return &utxo_coin.UtxoCoin{
			CoinConfig:     utxo_coin.CoinConfigs[utxo_coin.RAVENCOIN],
			PublicKeyBytes: btcecPubKey.SerializeCompressed(),
			Address:        fromAddress,
		}, nil
	case "VET":
		return vechain.NewVeChain(pubkeySlice, vechain.Mainnet), nil
	case "ZEC":
		btcecPubKey, err := btcec.ParsePubKey(pubkeySlice, btcec.S256())
		if err != nil {
			return nil, err
		}
		utxo := &utxo_coin.UtxoCoin{
			CoinConfig: utxo_coin.CoinConfigs[utxo_coin.ZCASH],
		}
		if _, err = utxo.GetAddress(btcecPubKey.SerializeCompressed()); err != nil {
			return nil, err
		}
		return utxo, nil
	case "XTZ":
		return &tezos.Tezos{PublicKey: pubkeySlice, Network: tezos.Mainnet}, nil
	case "NEO":
		return &neo.Neo{PublicKey: pubkeySlice, CoinConfig: utxo_coin.CoinConfigs[utxo_coin.NEO]}, nil
	case "XLM":
		return &stellar.Stellar{PublicKeyBytes: pubkeySlice}, nil
	case "EOS", "EOSPUBKEY":
		return &eos.Eos{}, nil
	case "POLYGON":
		if useTestnet {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Mumbai], ChainId[Mumbai]), nil
		} else {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Polygon], ChainId[Polygon]), nil
		}
	case "AVALANCHE":
		if useTestnet {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[AvalancheTestnet], ChainId[AvalancheTestnet]), nil
		} else {
			return ethereum.NewCustomEthereumStandard(RpcEndpoint[Avalanche], ChainId[Avalanche]), nil
		}
	case "SOL":
		if useTestnet {
			return solana.NewSolana(pubkeySlice, solana.Testnet), nil
		} else {
			return solana.NewSolana(pubkeySlice, solana.Mainnet), nil
		}
	case "TRON":
		if useTestnet {
			return &tron.Tron{PublicKey: pubkey, Network: tron.Testnet}, nil
		}
		return &tron.Tron{PublicKey: pubkey, Network: tron.Mainnet}, nil
	default:
		return nil, fmt.Errorf("network is not supported by server yet")
	}
}

func NativeToken(network string) string {
	switch network {
	case "BSC", "BNB":
		return "BNB"
	case "ETH":
		return "ETH"
	case "POLYGON":
		return "MATIC"
	case "AVALANCHE":
		return "AVAX"
	case "TRON":
		return "TRX"
	default:
		return network
	}
}

func NativeTokenDecimals(network string) int32 {
	switch network {
	case "BSC", "ETH", "POLYGON", "AVALANCHE":
		return 18
	case "SOL":
		return 9
	case "BNB", "BTC":
		return 8
	default:
		return 0
	}
}

func GetNetworkFromChainId(chainId int64, useTestnet bool) (string, error) {
	var network string
	switch chainId {
	case ChainId[Ethereum]:
		network = "ETH"
		if useTestnet {
			return "", errors.New("server is on testnet")
		}
	case ChainId[Ropsten]:
		network = "ETH"
		if !useTestnet {
			return "", errors.New("server is on mainnet")
		}
	case ChainId[Bsc]:
		network = "BSC"
		if useTestnet {
			return "", errors.New("server is on testnet")
		}
	case ChainId[BscTestnet]:
		network = "BSC"
		if !useTestnet {
			return "", errors.New("server is on mainnet")
		}
	case ChainId[Polygon]:
		network = "POLYGON"
		if useTestnet {
			return "", errors.New("server is on testnet")
		}
	case ChainId[Mumbai]:
		network = "POLYGON"
		if !useTestnet {
			return "", errors.New("server is on mainnet")
		}
	case ChainId[Avalanche]:
		network = "AVALANCHE"
		if useTestnet {
			return "", errors.New("server is on testnet")
		}
	case ChainId[AvalancheTestnet]:
		network = "AVALANCHE"
		if !useTestnet {
			return "", errors.New("server is on mainnet")
		}
	default:
		return "", fmt.Errorf("chainId %d is not supported", chainId)
	}
	return network, nil
}

func Network2CurveNetwork(network string) string {
	switch network {
	case "NEO", "ONT":
		return "ONT"
	case "XLM", "SOL": //XLM, SOL both use Ed25519 curve
		return "XLM"
	default:
		return "BNB"
	}
}

func Network2Curve(network string) elliptic.Curve {
	switch network {
	case "NEO", "ONT":
		return elliptic.P256()
	case "XLM", "SOL":
		return edwards.Edwards()
	default:
		return btcec.S256()
	}
}

func CurveIdToNetwork(curveId string) string {
	switch curveId {
	case "secp256k1":
		return "BNB"
	case "ed25519":
		return "XLM"
	default:
		return ""
	}
}
