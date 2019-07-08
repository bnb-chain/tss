#!/usr/bin/env bash

t=$1
n=$2
bnbcli=/Users/zhaocong/go/src/github.com/binance-chain/node/build/tbnbcli
bnbcli_home=~/.bnbcli_tss
tt=$(date '+%s')

rm -rf ./configs
./keygen.sh $1 $2 ./configs
sleep 30
./prepare_sign.sh $t $tt

mv ./configs/ ./configs_${tt}/

rm -rf $bnbcli_home
address=$($bnbcli --home $bnbcli_home keys add --tss -t tss --tss-home /Users/zhaocong/Developer/tss/configs_${tt}/0/ party0 | grep "party" | awk '{print $3}')
echo $address
touch ./configs_${tt}/$address
$bnbcli send --amount 1000000:BNB --to ${address} --from zcc --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node

for ((i=1; i<=$t; i++ ))
do
$bnbcli --home $bnbcli_home keys add --tss -t tss --tss-home ./configs_${tt}/$i/ party$i
done

for i in {0..20}
do
    mkdir -p ./configs_${tt}/signlog/$i/
    ./kill_all.sh
    ./sign.sh $t ./configs_${tt}/ ${bnbcli} ${bnbcli_home} $i
    sleep 30
done

sequence=$(curl https://testnet-dex.binance.org/api/v1/account/${address} | sed -n 's/.*sequence\":\(.*\)\}$/\1/p')
if [ "$sequence" == "21" ]; then
    echo "good"
else
    echo $'\n'"./configs_${tt}/" >> bad.txt
fi;