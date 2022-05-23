package blockchainUtils

type Network int

const (
	Ethereum Network = iota
	Ropsten
	Bsc
	BscTestnet
	Polygon
	Mumbai
	Avalanche
	AvalancheTestnet
)

var ChainId = map[Network]int64{
	Ethereum:         1,
	Ropsten:          3,
	Bsc:              56,
	BscTestnet:       97,
	Polygon:          137,
	Mumbai:           80001,
	Avalanche:        43114,
	AvalancheTestnet: 43113,
}

var RpcEndpoint = map[Network]string{
	Ethereum:         "https://mainnet.infura.io/v3/d4d164391311401b9ac5e900b0d9f7df",
	Ropsten:          "https://ropsten.infura.io/v3/a1ebd19437794205a2916e18e61394ef",
	Bsc:              "https://bsc-dataseed.binance.org/",
	BscTestnet:       "https://data-seed-prebsc-1-s1.binance.org:8545/",
	Polygon:          "https://polygon-rpc.com",
	Mumbai:           "https://polygon-mumbai.g.alchemy.com/v2/2RsRKQJiNfRh8vXh_9HwZLWNQVvE6feD",
	Avalanche:        "https://api.avax.network/ext/bc/C/rpc",
	AvalancheTestnet: "https://api.avax-test.network/ext/bc/C/rpc",
}
