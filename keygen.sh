#!/usr/bin/env bash

# usage ./keygen_36.sh t n home
t=$1
n=$2
home=$3

go build
echo "start server"
tss --home="./network/server" --p2p.listen="/ip4/127.0.0.1/tcp/27148" --mode="server" > server.log 2>&1 &
echo "setup"
tss --p2p.bootstraps="/ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1" --mode="setup" --threshold=$1 --parties=$2 > setup.log 2>&1
sleep 5

home=$3
for (( i=0; i<$2; i++ ))
do
    echo "start $i client"
    ((port=27150+$i))
    tss --home="${home}/${i}/" --p2p.listen="/ip4/127.0.0.1/tcp/${port}" --password "1234qwerasdf" > ${home}/client${i}.log 2>&1 &
    sleep 5
done