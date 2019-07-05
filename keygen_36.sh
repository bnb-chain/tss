#!/usr/bin/env bash

go build
rm -rf ./configs
echo "start server"
./tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
echo "setup"
./tss --p2p.bootstraps="/ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1" --mode="setup" --threshold=3 --parties=6 > setup.log 2>&1
sleep 5
echo "start first client"
./tss --home="./configs/0/" --p2p.listen="/ip4/127.0.0.1/tcp/27149" --profile_addr="127.0.0.1:6060" --password "1234qwerasdf" > client0.log  2>&1 &
sleep 5
echo "start second client"
./tss --home="./configs/1/" --p2p.listen="/ip4/127.0.0.1/tcp/27150" --profile_addr="127.0.0.1:6061" --password "1234qwerasdf"  > client1.log 2>&1 &
sleep 5
echo "start third client"
./tss --home="./configs/2/" --p2p.listen="/ip4/127.0.0.1/tcp/27151" --profile_addr="127.0.0.1:6062" --password "1234qwerasdf" > client2.log 2>&1 &
sleep 5
echo "start forth client"
./tss --home="./configs/3/" --p2p.listen="/ip4/127.0.0.1/tcp/27152" --profile_addr="127.0.0.1:6063" --password "1234qwerasdf" > client3.log 2>&1 &
sleep 5
echo "start fifth client"
./tss --home="./configs/4/" --p2p.listen="/ip4/127.0.0.1/tcp/27153" --profile_addr="127.0.0.1:6064" --password "1234qwerasdf" > client4.log 2>&1 &
sleep 5
echo "start sixth client"
./tss --home="./configs/5/" --p2p.listen="/ip4/127.0.0.1/tcp/27154" --profile_addr="127.0.0.1:6065" --password "1234qwerasdf" > client5.log 2>&1 &
