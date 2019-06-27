#!/usr/bin/env bash

echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
sleep 5
echo "start first client"
./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --mode="sign" --signers="party0,party1" --indexes="1,2" --message="42283978342796294123752210674145638809881570899772685773448437603618047389319" --password "1234qwerasdf" > client0_sign.log 2>&1 &
sleep 5
echo "start second client"
./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --mode="sign" --signers="party0,party1" --indexes="1,2" --message="42283978342796294123752210674145638809881570899772685773448437603618047389319" --password "1234qwerasdf" > client1_sign.log 2>&1 &
