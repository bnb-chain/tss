#!/usr/bin/env bash

go build -tags=deluxe
for (( i=0; i<=100; i++ ))
do
    echo "start $i client"
    mkdir ./logs/$i
    ./tss sign --home /Users/zhaocong/.test1/ --vault_name default --password 123456789 --channel_id 2585D860B19 --channel_password 123456789 --amount 10 --to bnb1ultyhpw2p2ktvr68swz56570lgj2rdsadq3ym2 --demon BNB --network Binance --broadcast true > ./logs/$i/first.log 2>&1 &
    ./tss sign --home /Users/zhaocong/.test2/ --vault_name default --password 123456789 --channel_id 2585D860B19 --channel_password 123456789 --amount 10 --to bnb1ultyhpw2p2ktvr68swz56570lgj2rdsadq3ym2 --demon BNB --network Binance > ./logs/$i/second.log 2>&1 &
    sleep 20
    ./kill_all.sh

done