#!/usr/bin/env bash

go build
rm -r ./configs
rm client0.log client1.log client2.log server.log setup.log
echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
echo "setup"
./tss --p2p.peer_addrs="/ip4/127.0.0.1/tcp/27149,/ip4/127.0.0.1/tcp/27150,/ip4/127.0.0.1/tcp/27151" --mode="setup" > setup.log 2>&1
sleep 5
echo "start first client"
./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" > client0.log --password "1234qwerasdf" 2>&1 &
sleep 5
echo "start second client"
./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" > client1.log --password "1234qwerasdf" 2>&1 &
sleep 5
echo "start third client"
./tss --home="./configs/2/" --p2p.listen="/ip4/127.0.0.1/tcp/27151" --profile_addr="127.0.0.1:6062" > client2.log --password "1234qwerasdf" 2>&1 &
