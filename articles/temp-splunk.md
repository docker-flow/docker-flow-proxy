```bash
docker network create --driver overlay proxy

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    -p 8089:8089 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    -e BIND_PORTS=8089 \
    docker-flow-proxy

docker service create --name splunk-reporter \
    -e SPLUNK_START_ARGS="--accept-license" \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath.1=/ \
    --label com.df.port.1=8089 \
    --label com.df.srcPort.1=443 \
    --label com.df.servicePath.2=/ \
    --label com.df.port.2=8000 \
    --label com.df.srcPort.2=80 \
    registry.splunk.com/splunk/splunk

docker service create --name splunk-reporter \
    -e SPLUNK_START_ARGS="--accept-license" \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/ \
    --label com.df.port=8089 \
    registry.splunk.com/splunk/splunk

curl "http://localhost:8080/v1/docker-flow-proxy/config"

curl -i "http://localhost"

curl -i "http://localhost:8089"

curl -i "https://localhost:8089"

curl -i -XPUT \
    --data-binary @tmp/xip.io.pem \
    "localhost:8080/v1/docker-flow-proxy/cert?certName=xip.io.pem&distribute=true"

curl -i --insecure "https://localhost:8089"

curl -i --insecure "https://192.168.1.35.xip.io:8089/"
```