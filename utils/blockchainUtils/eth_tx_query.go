package blockchainUtils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
)

const (
	Tx_Status_Success = 1
	Tx_Status_Fail    = 0
)

type EthereumRPC struct {
	Jsonrpc string   `json:"jsonrpc"`
	Id      int      `json:"id"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}

type RpcResponse struct {
	Jsonrpc string                 `json:"jsonrpc"`
	Id      int                    `json:"id"`
	Result  map[string]interface{} `json:"result"`
	Error   Error                  `json:"error"`
}

type BalanceRpcResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type TestnetFlag int

const (
	Mainnet TestnetFlag = iota
	Testnet
)

type NetworkKey struct {
	ChainNetwork string
	TestnetFlag
}

var accessPoint = map[NetworkKey]string{
	{"BSC", Mainnet}:     "https://bsc-dataseed.binance.org/",
	{"BSC", Testnet}:     "https://data-seed-prebsc-1-s1.binance.org:8545/",
	{"ETH", Mainnet}:     "https://mainnet.infura.io/v3/d4d164391311401b9ac5e900b0d9f7df",
	{"ETH", Testnet}:     "https://ropsten.infura.io/v3/a1ebd19437794205a2916e18e61394ef",
	{"BTC", Mainnet}:     "https://blockbook-bitcoin.binancechain.io",
	{"BTC", Testnet}:     "https://testnet.ejeokvv.com:19130",
	{"BCH", Mainnet}:     "https://blockbook-bcash.binancechain.io",
	{"BCH", Testnet}:     "https://testnet.ejeokvv.com:19131",
	{"LTC", Mainnet}:     "https://blockbook-litecoin.binancechain.io",
	{"LTC", Testnet}:     "https://testnet.ejeokvv.com:19134",
	{"DOGE", Mainnet}:    "https://us-doge2.twnodes.com",
	{"QTUM", Mainnet}:    "https://blockv3.qtum.info",
	{"RVN", Mainnet}:     "https://blockbook.ravencoin.org",
	{"DASH", Mainnet}:    "https://blockbook-dash.binancechain.io",
	{"ZEC", Mainnet}:     "https://blockbook-zcash.binancechain.io",
	{"POLYGON", Mainnet}: "https://polygon-rpc.com",
	{"TRON", Mainnet}:    "https://api.trongrid.io",
	{"TRON", Testnet}:    "https://api.shasta.trongrid.io",
}

var chainBlockBookURLMap = map[string]string{
	"BSC":     "https://blockbook-1.nariox.org",
	"POLYGON": "https://polygon.4983c65be9d9.com",
	"ETH":     "https://blockbook-eth.ninicoin.io:443",
}

type BlockBookBalanceResponse struct {
	Page        int `json:"page"`
	TotalPages  int `json:"totalPages"`
	ItemsOnPage int `json:"itemsOnPage"`
	Tokens      []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Contract  string `json:"contract"`
		Transfers int    `json:"transfers"`
		Symbol    string `json:"symbol"`
		Decimals  int    `json:"decimals"`
		Balance   string `json:"balance"`
	} `json:"tokens"`
}

func AccountGetBalance(address, network string) (map[string]string, error) {
	prefixURL := chainBlockBookURLMap[network]
	url := fmt.Sprintf("%v/api/v2/tokens/%v?details=balance", prefixURL, address)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusOK {
		if string(payload) == "[]\n" {
			return nil, nil
		}
		var retInfo = BlockBookBalanceResponse{}
		err := json.Unmarshal(payload, &retInfo)
		if err != nil {
			return nil, err
		}
		retMap := make(map[string]string)
		for _, v := range retInfo.Tokens {
			retMap[v.Symbol] = v.Balance
		}
		return retMap, nil
	} else {
		return nil, fmt.Errorf("failed to fetch account balance, status: %d, response: %s", res.StatusCode, string(payload))
	}
}

func EthGetContractAddress(tx string, useTestnet bool, chainNetwork string) (string, error) {
	reqPayload := EthereumRPC{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "eth_getTransactionReceipt",
		Params:  []string{tx},
	}
	jsonPayload, err := json.Marshal(&reqPayload)
	if err != nil {
		return "", err
	}

	network := Mainnet
	if useTestnet {
		network = Testnet
	}
	res, err := http.Post(accessPoint[NetworkKey{chainNetwork, network}], "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return "", err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		var accountInfo = RpcResponse{}
		err := json.Unmarshal(payload, &accountInfo)
		if err != nil {
			return "", err
		}
		logs, ok := accountInfo.Result["logs"].([]interface{})
		if !ok {
			return "", fmt.Errorf("failed to fetch contract address, response: %s", payload)
		}
		log, ok := logs[0].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("failed to fetch contract address, response: %s", payload)
		}
		contractAddress, ok := log["address"].(string)
		if !ok {
			return "", fmt.Errorf("failed to fetch contract address, response: %s", payload)
		}
		return contractAddress, nil
	} else {
		return "", fmt.Errorf("failed to fetch account nonce, status: %d, response: %s", res.StatusCode, string(payload))
	}
}

func ETHGetCode(address string, useTestnet bool, chainNetwork string) (int, error) {
	reqPayload := EthereumRPC{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "eth_getCode",
		Params:  []string{address, "latest"},
	}
	jsonPayload, err := json.Marshal(&reqPayload)
	if err != nil {
		return 0, err
	}

	network := Mainnet
	if useTestnet {
		network = Testnet
	}
	res, err := http.Post(accessPoint[NetworkKey{chainNetwork, network}], "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return 0, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	if res.StatusCode == http.StatusOK {
		var accountInfo = make(map[string]interface{})
		err := json.Unmarshal(payload, &accountInfo)
		if err != nil {
			return 0, err
		}
		return len(accountInfo["result"].(string)), nil
	} else {
		return 0, fmt.Errorf("failed to fetch account nonce, status: %d, response: %s", res.StatusCode, string(payload))
	}

}

func ETHGetTxStatus(tx string, useTestnet bool, chainNetwork string) (int, error) {
	reqPayload := EthereumRPC{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "eth_getTransactionReceipt",
		Params:  []string{tx},
	}
	jsonPayload, err := json.Marshal(&reqPayload)
	if err != nil {
		return Tx_Status_Fail, err
	}

	network := Mainnet
	if useTestnet {
		network = Testnet
	}
	res, err := http.Post(accessPoint[NetworkKey{chainNetwork, network}], "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return Tx_Status_Fail, err
	}
	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Tx_Status_Fail, err
	}
	if res.StatusCode == http.StatusOK {
		var accountInfo = RpcResponse{}
		err := json.Unmarshal(payload, &accountInfo)
		if err != nil {
			return Tx_Status_Fail, err
		}
		status, ok := accountInfo.Result["status"].(string)
		if !ok {
			return Tx_Status_Fail, fmt.Errorf("failed to fetch contract address, response: %s", payload)
		}
		if status == "0x1" {
			return Tx_Status_Success, nil
		}
		log.Debug().Msgf("query status = %s, accountInfo = %v", status, accountInfo)
		return Tx_Status_Fail, nil
	} else {
		return Tx_Status_Fail, fmt.Errorf("failed to fetch account nonce, status: %d, response: %s", res.StatusCode, string(payload))
	}
}

func ETXTxIsConfirmed(tx string, useTestnet bool, chainNetwork string) (bool, error) {
	statusCode, err := ETHGetTxStatus(tx, useTestnet, chainNetwork)
	return statusCode == Tx_Status_Success, err
}

// return true if input is a contract address
func IsContractAddress(address string, useTestnet bool, chainNetwork string) (bool, error) {
	length, err := ETHGetCode(address, useTestnet, chainNetwork)
	if err != nil {
		return false, err
	}
	// default length is "0x"
	if length > 2 {
		return true, nil
	}
	return false, nil
}
