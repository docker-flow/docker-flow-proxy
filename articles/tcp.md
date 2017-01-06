```bash
docker-machine create -d virtualbox tests

eval $(docker-machine env tests)

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    -p 6379:6379 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy:beta

docker service create --name redis \
    --network proxy \
    redis:3.2

docker service ps redis

curl "http://$(docker-machine ip tests):8080/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&srcPort=6379&reqMode=tcp"

docker service rm proxy redis
```