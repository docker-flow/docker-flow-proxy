Setup
===========

```bash
docker-machine create \
    -d virtualbox \
    docker-flow

eval "$(docker-machine env docker-flow)"

export DOCKER_IP=$(docker-machine ip docker-flow)

export CONSUL_IP=$(docker-machine ip docker-flow)

docker run -d \
    -p "8500:8500" \
    -h "consul" \
    --name "consul" \
    progrium/consul -server -bootstrap

docker run -d \
    --name registrator \
    -v /var/run/docker.sock:/tmp/docker.sock \
    -h $DOCKER_IP \
    gliderlabs/registrator \
    -ip $DOCKER_IP consul://$CONSUL_IP:8500
```

HAProxy
=======

```bash
go test --cover

docker run --rm \
    -v $PWD:/usr/src/myapp \
    -w /usr/src/myapp \
    -v $GOPATH:/go \
    golang:1.6 \
    go build -v -o docker-flow-proxy

docker build -t vfarcic/docker-flow-proxy .

# Start HA

docker rm -f docker-flow-proxy

docker run -d \
    --name docker-flow-proxy \
    -e CONSUL_ADDRESS=${CONSUL_IP}:8500 \
    -p 80:80 \
    vfarcic/docker-flow-proxy

# Start the service

docker-compose up -d app db

# Update HA

docker exec docker-flow-proxy \
    docker-flow-proxy \
    reconfigure --service-name books-ms --service-path /api/v1/books

curl -I $DOCKER_IP/api/v1/books
```
