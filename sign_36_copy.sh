#!/usr/bin/env bash

echo "start server"
tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 10000:BNB --to tbnb1ldyr87kk8705dzk9tpqhrua2y43m6ucwkyela8 --from party360 --chain-id test-chain-n4b735 > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 10000:BNB --to tbnb1ldyr87kk8705dzk9tpqhrua2y43m6ucwkyela8 --from party362 --chain-id test-chain-n4b735 > client1_sign.log 2>&1 &
sleep 5
echo "start third client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 10000:BNB --to tbnb1ldyr87kk8705dzk9tpqhrua2y43m6ucwkyela8 --from party364 --chain-id test-chain-n4b735 > client4_sign.log 2>&1 &
sleep 5
echo "start forth client"
/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli send --amount 10000:BNB --to tbnb1ldyr87kk8705dzk9tpqhrua2y43m6ucwkyela8 --from party365 --chain-id test-chain-n4b735 > client5_sign.log 2>&1 &