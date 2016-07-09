```
# Infra Setup

git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

docker-machine create -d virtualbox node-1

docker-machine create -d virtualbox node-2

docker-machine create -d virtualbox node-3

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

# Deployment

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

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    --constraint node.id==$(docker node inspect node-1 --format "{{.ID}}") \
    -e MODE=service \
    vfarcic/docker-flow-proxy

docker service ls # Repeat until proxy REPLICAS is set to 1/1

docker service tasks proxy

curl $(docker-machine ip node-1)/demo/hello

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080"

curl $(docker-machine ip node-1)/demo/hello

docker service update --replicas 5 go-demo

docker service ls

docker service tasks go-demo

curl $(docker-machine ip node-1)/demo/hello
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
