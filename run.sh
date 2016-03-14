#!/usr/bin/env bash

set -e

docker-flow-proxy $@

if [[ "$1" == "run" ]]; then
    sleep infinity
fi