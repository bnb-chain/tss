#!/usr/bin/env bash

t=$1
homeSuffix=$2

signers="[\"party0\""
for (( i=1; i<=$t; i++ ))
do
    signers="$signers, \"party$i\""
done
signers="$signers]"

for (( i=0; i<=$t; i++ ))
do
    ((port=27150+$i))
    sed -i -e "s/ip4\/0.0.0.0\/tcp\/27148/ip4\/0.0.0.0\/tcp\/$port/g" ./configs/${i}/config.json
    sed -i -e "s/\/Users\/zhaocong\/.tss/\/Users\/zhaocong\/Developer\/tss\/configs_${homeSuffix}\/$i/g" ./configs/${i}/config.json
    sed -i -e "s/keygen/sign/g" ./configs/${i}/config.json
    sed -i -e "s/\"Signers\": null/\"Signers\": $signers/g" ./configs/${i}/config.json
done