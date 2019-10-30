tss
---

Cli and transportation wrapper of [tss-lib](https://github.com/binance-chain/tss-lib)

## Build
Ths transaction building is delegated to [trust-wallet-core](https://github.com/trustwallet/wallet-core). Thus you should install [pre-requirement](https://developer.trustwallet.com/wallet-core/contributing#before-you-start) of trust-wallet-core.

After you installed above pre-requirement, you could run `make build` to bulid a tss executable support sign command.

## Play in localhost

Note: `--password` option should only be used in testing. 

Without command line option, the cli would ask interactive input and confirm.

1. init 3 parties
```
./tss init --home ~/.test1 --vault_name "default" --moniker "test1" --password "123456789"
./tss init --home ~/.test2 --vault_name "default" --moniker "test2" --password "123456789"
./tss init --home ~/.test3 --vault_name "default" --moniker "test3" --password "123456789"
```

2. generate channel id
replace value of "--channel_id" for following commands with generated one
```
./tss channel --channel_expire 30
```

3. keygen 
```
./tss keygen --home ~/.test1 --vault_name "default" --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "802671B1B19"   
./tss keygen --home ~/.test2 --vault_name "default" --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "802671B1B19"            
./tss keygen --home ~/.test3 --vault_name "default" --parties 3 --threshold 1 --password "123456789" --channel_password "123456789" --channel_id "802671B1B19" 
```

4. sign
```
./tss sign --home ~/.test1 --vault_name default --password 123456789 --channel_id 802671B1B19 --channel_password 123456789 --amount 10 --to 0xe9C204d1b24adB1205D4947b1fa8d6CBf9AF0d32 --demon ETH --network EthereumRopsten --broadcast
./tss sign --home ~/.test2 --vault_name default --password 123456789 --channel_id 802671B1B19 --channel_password 123456789 --amount 10 --to 0xe9C204d1b24adB1205D4947b1fa8d6CBf9AF0d32 --demon ETH --network EthereumRopsten
```

5. regroup - replace existing 3 parties with 3 brand new parties
```
# start 2 old parties (answer Y for isOld and IsNew interactive questions)
./tss regroup --home ~/.test1 --vault_name "default" --password "123456789" --new_parties 3 --new_threshold 1 --channel_password "123456789" --channel_id "802671B1B19"
./tss regroup --home ~/.test2 --vault_name "default" --password "123456789" --new_parties 3 --new_threshold 1 --channel_password "123456789" --channel_id "802671B1B19"
# start the new parties (answer n for isIold and Y for IsNew interactive questions)
./tss regroup --home ~/.test3 --vault_name "default" --password "123456789" --new_parties 3 --new_threshold 1 --channel_password "123456789" --channel_id "802671B1B19"
```

## Network roles and connection topological
![](network/tss.png)

## Supported NAT Types

Referred to https://github.com/libp2p/go-libp2p/issues/375#issuecomment-407122416 We also have three nat-traversal solutions at the moment.

1. UPnP/NATPortMap 
<br><br> When NAT traversal is enabled (in go-libp2p, pass the NATPortMap() option to the libp2p constructor), libp2p will use UPnP and NATPortMap to ask the NAT's router to open and forward a port for libp2p. If your router supports either UPnP or NATPortMap, this is by far the best option.

2. STUN/hole-punching
<br><br> LibP2P has it's own version of the "STUN" protocol using peer-routing, external address discovery, and reuseport.

3. TURN-like protocol (relay)
<br><br> Finally, we have a TURN like protocol called p2p-circuit. This protocol allows libp2p nodes to "proxy" through other p2p nodes. All party clients registered to mainnet would automatically announce they support p2p-circuit (relay) for tss implementation.



### In WAN setting

| | Full cone | (Address)-restricted-cone | Port-restricted cone	| Symmetric NAT |
| ------ | ------ | ------ | ------ | ------ |
|Bootstrap (tracking) server| ✓ | ✘ | ✘ | ✘ |
|Relay server| ✓ | ✘ | ✘ | ✘ |
|Client| ✓ | ✓ | ✓ | ✓ (relay server needed) |

### In LAN setting

Nodes can connected to each other directly without setting bootstrap and relay server.  
We have 3 layers of bootstrapping session to help nodes connect with each other within a LAN
1. ssdp - started before 2 (raw tcp bootstrapping), node advertise their listen addr and moniker and record others. This is not encrypted.
2. raw tcp bootstrapping - node connect with each other via raw tcp to communicate their libp2pid, moniker, listen address. This is encrypted with channel id and channel password.
3. libp2p - node share signers/whether it is new party in regroup via formal libp2p
Note: keygen and regroup would relies on 1,2,3. But sign only relies on 3, which means the sign can achieved in WAN (with bootstrap server's help)
