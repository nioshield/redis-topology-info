# redis-topology-info
Print redis info base on [redis-topology](https://github.com/shafreeck/redis-topology)

## Features

* Recursively print slave info 
* Support redis info section 

## redis.txt

List host and port in files each line, separated host and port by space.

```
host1 port1
host2 port2
host3 port3
host4 port4
```

## Usage

```
cat redis.txt | redis-topology-info -a <auth> memory mem_fragmentation_ratio
```

## Support command
- memory
- replication 
- server
- persistence 
- stats 
- cpu 
- cluster 
- keyspace
```
```
```
```
