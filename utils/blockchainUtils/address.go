package blockchainUtils

import (
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/btcsuite/btcd/btcec"
)

func Pubkey2Address(network, pubkey string, useTestnet bool) (string, error) {
	chain, err := InitBlockchain(network, "", pubkey, useTestnet)
	if err != nil {
		return "", err
	}
	switch network {
	case "ONT", "NEO":
		pubkeySlice, err := hex.DecodeString(pubkey)
		if err != nil {
			return "", fmt.Errorf("failed to decode public key on signRequest, original address message might be invalid")
		}
		return chain.GetAddress(pubkeySlice)
	case "ETH", "BSC", "ETC", "VET", "POLYGON", "AVALANCHE":
		byte, err := hex.DecodeString(pubkey)
		if err != nil {
			return "", err
		}
		btcecPubKey, err := btcec.ParsePubKey(byte, btcec.S256())
		bytePubKey := btcecPubKey.SerializeUncompressed()
		return chain.GetAddress(bytePubKey)
	case "BTCSEGWIT":
		byte, err := hex.DecodeString(pubkey)
		if err != nil {
			return "", err
		}
		return chain.GetAddress(byte)
	case "EOS":
		// please create EOSPUBKEY and create account on https://eos-account-creator.com/choose/, then add it
		return "", fmt.Errorf("please create EOS pubkey first")
	default:
		byte, err := hex.DecodeString(pubkey)
		if err != nil {
			return "", err
		}
		return chain.GetAddress(byte)
	}
}

func ValidateToAddress(network, address string, useTestnet bool) error {
	if network == "BTC" || network == "BTCSEGWIT" {
		if ValidateAddress("BTC", address, useTestnet) == nil || ValidateAddress("BTCSEGWIT", address, useTestnet) == nil {
			return nil
		}
		return fmt.Errorf("invalid address for network %s", network)
	} else {
		return ValidateAddress(network, address, useTestnet)
	}
}

func ValidateAddress(network, address string, useTestnet bool) error {
	var result bool
	var err error
	switch network {
	case "BNB":
		if useTestnet {
			result, err = regexp.Match(`^(tbnb1)[0-9a-z]{38}$`, []byte(address))
			if err != nil {
				return err
			}
		} else {
			result, err = regexp.Match(`^(bnb1)[0-9a-z]{38}$`, []byte(address))
			if err != nil {
				return err
			}
		}
	case "ETH", "BSC", "ETC", "VET", "POLYGON", "AVALANCHE":
		result, err = regexp.Match(`^(0x){1}[0-9a-fA-F]{40}$`, []byte(address))
		if err != nil {
			return err
		}
	case "BTC":
		if useTestnet {
			result, err = regexp.Match(`^[2nm][a-km-zA-HJ-NP-Z1-9]{25,34}$`, []byte(address))
			if err != nil {
				return err
			}
		} else {
			result, err = regexp.Match(`^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$`, []byte(address))
			if err != nil {
				return err
			}
		}
	case "BTCSEGWIT":
		if useTestnet {
			result, err = regexp.Match(`^(tb)[0-9A-Za-z]{39,59}$`, []byte(address))
			if err != nil {
				return err
			}
		} else {
			result, err = regexp.Match(`^(bc1)[0-9A-Za-z]{39,59}$`, []byte(address))
			if err != nil {
				return err
			}
		}
	case "BCH":
		result, err = regexp.Match(`^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^[0-9A-Za-z]{42,42}$`, []byte(address))
		if err != nil {
			return err
		}
	case "ATOM":
		result, err = regexp.Match(`^(cosmos1)[0-9a-z]{38}$`, []byte(address))
		if err != nil {
			return err
		}
	case "DASH":
		result, err = regexp.Match(`^[X|7][0-9A-Za-z]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	case "DOGE":
		result, err = regexp.Match(`^(D|A|9)[a-km-zA-HJ-NP-Z1-9]{33,34}$`, []byte(address))
		if err != nil {
			return err
		}
	case "ICX":
		result, err = regexp.Match(`^(hx)[A-Za-z0-9]{40}$`, []byte(address))
		if err != nil {
			return err
		}
	case "LTC":
		if useTestnet {
			result, err = regexp.Match(`^[2nm][a-km-zA-HJ-NP-Z1-9]{25,34}$`, []byte(address))
			if err != nil {
				return err
			}
		} else {
			result, err = regexp.Match(`^(L|M|3)[A-Za-z0-9]{33}$|^(ltc1)[0-9A-Za-z]{39}$`, []byte(address))
			if err != nil {
				return err
			}
		}
	case "ONT":
		result, err = regexp.Match(`^(A)[A-Za-z0-9]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	case "QTUM":
		result, err = regexp.Match(`^[Q|M][A-Za-z0-9]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	case "RVN":
		result, err = regexp.Match(`^[Rr]{1}[A-Za-z0-9]{33,34}$`, []byte(address))
		if err != nil {
			return err
		}
	case "XRP":
		result, err = regexp.Match(`^r[1-9A-HJ-NP-Za-km-z]{25,34}$`, []byte(address))
		if err != nil {
			return err
		}
	case "ZEC":
		result, err = regexp.Match(`^(t)[A-Za-z0-9]{34}$`, []byte(address))
		if err != nil {
			return err
		}
	case "XTZ":
		result, err = regexp.Match(`^(tz[1,2,3]|KT1)[a-zA-Z0-9]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	case "NEO":
		result, err = regexp.Match(`^(A)[A-Za-z0-9]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	case "XLM":
		result, err = regexp.Match(`^G[A-D]{1}[A-Z2-7]{54}$`, []byte(address))
		if err != nil {
			return err
		}
	case "EOSPUBKEY":
		result, err = regexp.Match(`^EOS[1-9A-HJ-NP-Za-km-z]{50}$`, []byte(address))
		if err != nil {
			return err
		}
	case "EOS":
		result, err = regexp.Match(`^[1-5a-z\\.]{1,12}$`, []byte(address))
		if err != nil {
			return err
		}
	case "SOL":
		result, err = regexp.Match(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`, []byte(address))
	case "TRON":
		result, err = regexp.Match(`^T[1-9A-HJ-NP-Za-km-z]{33}$`, []byte(address))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("network %s is unsupported", network)
	}
	if !result {
		return fmt.Errorf("invalid address for network %s", network)
	}
	return nil
}
