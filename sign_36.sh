#!/usr/bin/env bash

go build
echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:ZCB-F00 --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party360 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:ZCB-F00 --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party361 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client1_sign.log 2>&1 &
sleep 5
echo "start third client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:ZCB-F00 --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party362 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client2_sign.log 2>&1 &
sleep 5
echo "start forth client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:ZCB-F00 --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party363 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client3_sign.log 2>&1 &