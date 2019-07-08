#!/usr/bin/env bash

go build
#echo "start server"
#./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
#sleep 5
echo "start first client"
#./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --mode="sign" --signers="party0,party1" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client0_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 100000000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party0 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
#./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --mode="sign" --signers="party0,party1" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client1_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 100000000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party1 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client1_sign.log 2>&1 &
