#!/bin/bash

STACK_NAME="proxy"
export CORE_PATH=`pwd`/$STACK_NAME

docker stack deploy -c ./proxy.yml $STACK_NAME
