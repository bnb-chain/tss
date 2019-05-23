#!/usr/bin/env bash

go build
rm -r ./configs
echo "start server"
./tss --node_key="./network/server/node_key" --listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
echo "setup"
./tss --bootstraps="/ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1" --mode="setup"
echo "start first client"
./tss --config_path="./configs/0/" --node_key="./configs/0/node_key" --route_table="./configs/0/" --listen="/ip4/127.0.0.1/tcp/27149" > client0.log 2>&1 &
echo "start second client"
./tss --config_path="./configs/1/" --node_key="./configs/1/node_key" --route_table="./configs/1/" --listen="/ip4/127.0.0.1/tcp/27150" > client1.log 2>&1 &
echo "start third client"
./tss --config_path="./configs/2/" --node_key="./configs/2/node_key" --route_table="./configs/2/" --listen="/ip4/127.0.0.1/tcp/27151" > client2.log 2>&1 &
