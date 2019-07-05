#!/usr/bin/env bash

go build
echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
#./tss --home="./configs_36/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --mode="sign" --signers="party0,party1,party2,party3" --indexes="4,3,0,5" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client0_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party360 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
#./tss --home="./configs_36/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --mode="sign" --signers="party0,party1,party2,party3" --indexes="4,3,0,5" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client1_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party361 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client1_sign.log 2>&1 &
sleep 5
echo "start third client"
#./tss --home="./configs_36/2/" --p2p.listen="/ip4/127.0.0.1/tcp/27151" --profile_addr="127.0.0.1:6062" --mode="sign" --signers="party0,party1,party2,party3" --indexes="4,3,0,5" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client2_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party362 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client2_sign.log 2>&1 &
sleep 5
echo "start forth client"
#./tss --home="./configs_36/3/" --p2p.listen="/ip4/127.0.0.1/tcp/27152" --profile_addr="127.0.0.1:6063" --mode="sign" --signers="party0,party1,party2,party3" --indexes="4,3,0,5" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client3_sign.log 2>&1 &
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from party363 --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > client3_sign.log 2>&1 &