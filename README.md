tss
---

Cli and transportation wrapper of [tss-lib](https://github.com/binance-chain/tss-lib)

## Play in localhost
1. init 3 parties
```
go build
./tss init --home ~/.test1 --p2p.listen "/ip4/0.0.0.0/tcp/27148" --moniker "test1"
./tss init --home ~/.test2 --p2p.listen "/ip4/0.0.0.0/tcp/27149" --moniker "test2"
./tss init --home ~/.test3 --p2p.listen "/ip4/0.0.0.0/tcp/27150" --moniker "test3"
```

2. generate channel id
replace value of "--channel_id" for following commands with generated one
```
./tss channel
```

3. keygen 
```
keygen --channel_id "2855D42A535" --channel_password "123456789" --home ~/.test1 --parties 3 --threshold 1 --p2p.peer_addrs "/ip4/127.0.0.1/tcp/27150,/ip4/127.0.0.1/tcp/27149" --password "123456789"
keygen --channel_id "2855D42A535" --channel_password "123456789" --home ~/.test2 --parties 3 --threshold 1 --p2p.peer_addrs "/ip4/127.0.0.1/tcp/27150,/ip4/127.0.0.1/tcp/27148" --password "123456789"
keygen --channel_id "2855D42A535" --channel_password "123456789" --home ~/.test3 --parties 3 --threshold 1 --p2p.peer_addrs "/ip4/127.0.0.1/tcp/27148,/ip4/127.0.0.1/tcp/27149" --password "123456789"
```

4. sign
```
sign --home ~/.test1 --password "123456789" --channel_id "2855D42A535" --channel_password "123456789"
sign --home ~/.test2 --password "123456789" --channel_id "2855D42A535" --channel_password "123456789"
```

5. regroup
```

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

Nodes can connected to each other directly without setting bootstrap and relay server. But host(ip)&port of expectedPeers should be configured.

Refer to `setup_without_bootstrap.sh` to see how LAN direct connection can be used.