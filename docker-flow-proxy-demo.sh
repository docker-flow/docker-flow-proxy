#!/usr/bin/env bash

git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

docker-machine create -d virtualbox proxy

export DOCKER_IP=$(docker-machine ip proxy)

export CONSUL_IP=$(docker-machine ip proxy)

export PROXY_IP=$(docker-machine ip proxy)

eval "$(docker-machine env proxy)"

docker-compose up -d consul proxy

docker ps -a

docker-machine create -d virtualbox \
    --swarm --swarm-master \
    --swarm-discovery="consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-store=consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    swarm-master

docker-machine create -d virtualbox \
    --swarm \
    --swarm-discovery="consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-store=consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    swarm-node-1

docker-machine create -d virtualbox \
    --swarm \
    --swarm-discovery="consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-store=consul://$CONSUL_IP:8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    swarm-node-2

eval "$(docker-machine env --swarm swarm-master)"

docker info

eval "$(docker-machine env swarm-master)"

export DOCKER_IP=$(docker-machine ip swarm-master)

docker-compose up -d registrator

eval "$(docker-machine env swarm-node-1)"

export DOCKER_IP=$(docker-machine ip swarm-node-1)

docker-compose up -d registrator

eval "$(docker-machine env swarm-node-2)"

export DOCKER_IP=$(docker-machine ip swarm-node-2)

docker-compose up -d registrator

eval "$(docker-machine env --swarm swarm-master)"

docker ps -a

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    up -d

docker ps -a

eval "$(docker-machine env proxy)"

docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books

curl -I $PROXY_IP/api/v1/books