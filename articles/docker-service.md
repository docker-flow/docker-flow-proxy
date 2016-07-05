```
# Infra Setup

git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

docker-machine create -d virtualbox node-1

eval $(docker-machine env node-1)

DOCKER_IP=$(docker-machine ip node-1) docker-compose up -d consul-server

docker-machine create -d virtualbox node-2

eval $(docker-machine env node-2)

DOCKER_IP=$(docker-machine ip node-2) \
    CONSUL_IP=$(docker-machine ip node-2) \
    CONSUL_SERVER_IP=$(docker-machine ip node-1) \
    docker-compose up -d consul-agent

docker-machine create -d virtualbox node-3

eval $(docker-machine env node-3)

DOCKER_IP=$(docker-machine ip node-3) \
    CONSUL_IP=$(docker-machine ip node-3) \
    CONSUL_SERVER_IP=$(docker-machine ip node-1) \
    docker-compose up -d consul-agent

# Swarm Setup

eval $(docker-machine env node-1)

docker --version

docker swarm init --listen-addr $(docker-machine ip node-1):2377

docker info

docker node ls

eval $(docker-machine env node-2)

docker swarm join $(docker-machine ip node-1):2377

eval $(docker-machine env node-3)

docker swarm join $(docker-machine ip node-1):2377

eval $(docker-machine env node-1)

docker node ls

docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker network ls

docker service create --name go-demo-db \
  -p 27017 \
  --network go-demo \
  mongo

docker service ls # Repeat until go-demo-db REPLICAS is set to 1/1

docker service inspect -p go-demo-db

docker service tasks go-demo-db

docker service create --name go-demo \
  -p 8080 \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  vfarcic/go-demo

docker service ls # Repeat until go-demo REPLICAS is set to 1/1

docker service tasks go-demo

export CONSUL_IP=$(docker-machine ip node-1)

docker service create --name docker-flow-proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    --constraint NAME:node-1 \
    -e CONSUL_ADDRESS=$CONSUL_IP:8500 \
    -e PROXY_INSTANCE_NAME=docker-flow \
    -e MODE=service \
    vfarcic/docker-flow-proxy

docker service ls # Repeat until docker-flow-proxy REPLICAS is set to 1/1

docker service tasks docker-flow-proxy

# docker-compose up -d proxy-service

curl $(docker-machine ip node-1)/demo/hello

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080"

docker service ls # Repeat until proxy REPLICAS is set to 1/1

docker service tasks proxy


docker service update --replicas 3 go-demo

docker service ls

docker service tasks go-demo

```




```bash
docker service create --name debug \
    --network proxy \
    --mode global \
    alpine sleep 1000000000

docker service create --name debug \
    --network go-demo \
    --mode global \
    alpine sleep 1000000000

docker service tasks debug

CID=$(docker ps -q --filter label=com.docker.swarm.service.name=debug)

docker exec -ti $CID sh

apk add --update curl apache2-utils drill

drill go-demo

curl go-demo:8080/demo/hello

exit

docker service rm debug
















./consul-template -consul $(docker-machine ip node-1):8500 -template "nodes.ctmpl:nodes.txt" -once
```

TODO
----

* Proxy

  * Note that sticky sessions will not work
  * Note that there is no need for Registrator

* Rolling updates (http://view.dckr.info:8080/#127)
* Failover
* Bundle
* SwarmKit (http://view.dckr.info:8080/#129)
* Logging (http://view.dckr.info:8080/#138)
