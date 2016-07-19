#!/usr/bin/env bash

set -e

docker-machine create -d virtualbox node-1

docker-machine create -d virtualbox node-2

docker-machine create -d virtualbox node-3

eval $(docker-machine env node-1)

docker swarm init \
    --secret my-secret \
    --auto-accept worker \
    --listen-addr $(docker-machine ip node-1):2377

eval $(docker-machine env node-2)

docker swarm join \
    --secret my-secret \
    $(docker-machine ip node-1):2377

eval $(docker-machine env node-3)

docker swarm join \
    --secret my-secret \
    $(docker-machine ip node-1):2377

eval $(docker-machine env node-1)