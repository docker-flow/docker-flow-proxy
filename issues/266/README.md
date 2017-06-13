```bash
docker network create -d overlay proxy

docker network create -d overlay proxy2

# Changed to vfarcic instead medialon (it was not available)
# Removed `node.hostname == swarm-node-master-1` constraints

docker stack deploy -c proxy.yml proxy

# Removed `node.role != manager` constraint
# Removed `node.hostname == swarm-node-0` constraint

docker stack deploy -c stack.yml stack

docker service logs proxy_proxy
```