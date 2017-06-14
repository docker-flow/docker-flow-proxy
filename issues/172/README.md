# Running in a single node cluster

* Changed to one replica of each service

```bash
docker network create --driver overlay proxy

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener

docker network create --driver overlay myapp

docker service create --name myapp \
    -e DB=go-demo-db \
    -p 1234:8081 \
    --network myapp \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/demo \
    --label com.df.reqPathSearch=/demo/ \
    --label com.df.reqPathReplace=/ \
    --label com.df.port=8081 \
    solarwinds/whd-embedded:latest

docker service ps myapp

curl "http://localhost/demo"

curl "http://localhost:1234/demo"

curl "http://myapp:8081/demo"
```