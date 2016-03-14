Setup
=====

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

docker-machine create \
    -d virtualbox \
    docker-flow

eval "$(docker-machine env docker-flow)"

export DOCKER_IP=$(docker-machine ip docker-flow)

export CONSUL_IP=$(docker-machine ip docker-flow)

docker-compose up -d
```

Single Instance
===============

```bash
docker-compose \
    -f docker-compose-demo.yml \
    up -d

curl -I $DOCKER_IP/api/v1/books

docker exec docker-flow-proxy \
    docker-flow-proxy \
    reconfigure --service-name books-ms --service-path /api/v1/books

curl -I $DOCKER_IP/api/v1/books
```

Multiple Instances
==================

```bash
docker-compose \
    -f docker-compose-demo.yml \
    scale app=3

docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books

curl -I $DOCKER_IP/api/v1/books
```

```bash
docker exec -it docker-flow-proxy \
    cat /cfg/haproxy.cfg
```
