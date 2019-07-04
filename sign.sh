#!/usr/bin/env bash

echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --mode="sign" --signers="party0,party1" --indexes="0,2" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --mode="sign" --signers="party0,party1" --indexes="0,2" --password "1234qwerasdf" --message="22834074397645376241779885292397614307459065419949970031674264572936783990563" > client1_sign.log 2>&1 &
