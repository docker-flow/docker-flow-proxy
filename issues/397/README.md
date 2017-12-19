```bash
./swarm-cluster.sh

eval $(docker-machine env node-1)

docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

curl -i "http://$(docker-machine ip node-1)/demo/hello" # Wait until 200

docker service ps proxy_proxy # Confirm that a replica is running on node-3. If not, change the commands listed below to use a different node

docker-machine rm -f node-3

# Wait a few moments until Swarm realizes that a node is destroyed and stops sending ingress traffic to it

curl -i "http://$(docker-machine ip node-1)/demo/hello"
```
