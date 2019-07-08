#!/usr/bin/env bash

t=$1
home=$2
bnbcli=$3
bnbcli_home=$4
iter=$5

go build
echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
for (( i=0; i<=$t; i++ ))
do
    echo "start $i client"
    $bnbcli --home ${bnbcli_home} send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party$i --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > ${home}/signlog/$iter/client${i}_sign.log 2>&1 &
    sleep 5
done