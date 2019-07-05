#!/usr/bin/env bash

for i in {0..5}
do
    ((port=27150+$i))
    sed -i -e "s/ip4\/0.0.0.0\/tcp\/27148/ip4\/0.0.0.0\/tcp\/$port/g" ./configs/${i}/config.json
    sed -i -e "s/\/Users\/zhaocong\/.tss/\/Users\/zhaocong\/Developer\/tss\/configs_36\/$i/g" ./configs/${i}/config.json
    sed -i -e "s/keygen/sign/g" ./configs/${i}/config.json
    sed -i -e "s/\"Signers\": null/\"Signers\": [\"party0\", \"party1\", \"party2\", \"party3\"]/g" ./configs/${i}/config.json
done