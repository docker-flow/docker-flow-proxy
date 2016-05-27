#!/usr/bin/env bash

set -e

docker-machine create -d virtualbox proxy

export DOCKER_IP=$(docker-machine ip proxy)

export CONSUL_IP=$(docker-machine ip proxy)

eval "$(docker-machine env proxy)"

docker-compose up -d consul-server

sleep 2

docker-compose up -d proxy registrator
