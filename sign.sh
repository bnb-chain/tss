#!/usr/bin/env bash

echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --mode="sign" --signers="party0,party1" --indexes="0,2" --message="73074187637213954569750759329253492137642477709348593499227426310712237152695" --password "1234qwerasdf" > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --mode="sign" --signers="party0,party1" --indexes="0,2" --message="73074187637213954569750759329253492137642477709348593499227426310712237152695" --password "1234qwerasdf" > client1_sign.log 2>&1 &
