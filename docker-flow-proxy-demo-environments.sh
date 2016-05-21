#!/usr/bin/env bash

set -e

docker-machine create -d virtualbox proxy

export DOCKER_IP=$(docker-machine ip proxy)

export CONSUL_IP=$(docker-machine ip proxy)

eval "$(docker-machine env proxy)"

docker-compose up -d consul-server

sleep 2

docker-compose up -d proxy

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

eval "$(docker-machine env swarm-master)"

export DOCKER_IP=$(docker-machine ip swarm-master)

docker-compose up -d registrator

eval "$(docker-machine env swarm-node-1)"

export DOCKER_IP=$(docker-machine ip swarm-node-1)

docker-compose up -d registrator

eval "$(docker-machine env swarm-node-2)"

export DOCKER_IP=$(docker-machine ip swarm-node-2)

docker-compose up -d registrator
