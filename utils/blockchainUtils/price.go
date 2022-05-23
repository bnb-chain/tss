package blockchainUtils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Cryptocurrency struct {
	Status struct {
		Timestamp time.Time `json:"timestamp"`
	} `json:"status"`
	Data []struct {
		Symbol string `json:"symbol"`
		Quote  struct {
			Usd struct {
				Price float64 `json:"price"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}

func getCryptocurrency() (*Cryptocurrency, error) {
	url := "https://wallet.binance.org/api/v1/cryptocurrency"
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fail to fetch crypto currency info: %v", err)
	}
	if res.StatusCode == http.StatusOK {
		payload, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("fail to read rpc response: %v", err)
		}
		var ccInfo Cryptocurrency
		err = json.Unmarshal(payload, &ccInfo)
		if err != nil {
			return nil, fmt.Errorf("fail to unmarshal crypto currency info: %v", err)
		} else {
			return &ccInfo, nil
		}
	} else {
		errPayload, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to fetch crypto currency info, rpc returns %d: %s", res.StatusCode, string(errPayload))
	}
}

func GetCryptocurrencyUSDPriceMap() (map[string]float64, error) {
	ccInfo, err := getCryptocurrency()
	if err != nil {
		return nil, err
	}
	priceMap := make(map[string]float64)
	for _, v := range ccInfo.Data {
		priceMap[v.Symbol] = v.Quote.Usd.Price
	}
	return priceMap, nil
}
