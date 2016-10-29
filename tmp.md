```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener

docker service create --name go-demo-db \
  --network go-demo \
  mongo

docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  vfarcic/go-demo



VERSION=beta

docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.6 bash -c "cd /usr/src/myapp && go get -d -v -t && go test --cover ./... --run UnitTest && go build -v -o docker-flow-proxy"

docker build -t vfarcic/docker-flow-proxy:${VERSION} .

docker push vfarcic/docker-flow-proxy:${VERSION}

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy:beta

docker service rm go-demo

docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  vfarcic/go-demo

curl -i localhost/demo/hello

docker service rm proxy
```