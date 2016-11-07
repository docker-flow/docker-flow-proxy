```bash
# https://serversforhackers.com/using-ssl-certificates-with-haproxy

mkdir -p tmp/xip.io

openssl genrsa -out tmp/xip.io/xip.io.key 1024

openssl req -new \
    -key tmp/xip.io/xip.io.key \
    -out tmp/xip.io/xip.io.csr

# ES
# Barcelona
# Barcelona
# TechnologyConversations.com
#
# *.xip.io
# viktor@farcic.com
#
#

openssl x509 -req -days 365 \
    -in tmp/xip.io/xip.io.csr \
    -signkey tmp/xip.io/xip.io.key \
    -out tmp/xip.io/xip.io.crt

cat tmp/xip.io/xip.io.crt tmp/xip.io/xip.io.key \
    | tee tmp/xip.io/xip.io.pem

docker network create --driver overlay proxy

docker network create --driver overlay go-demo

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
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy

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
    --label com.df.serviceDomain=xip.io \
    vfarcic/go-demo

docker service ls

# Find out the IP

IP=192.168.1.36

curl -i http://$IP.xip.io/demo/hello

curl -i https://$IP.xip.io/demo/hello

# TODO: Remove

docker service update \
    --publish-add 8080:8080 proxy

# TODO: Remove

docker service inspect proxy --pretty

curl localhost:8080/v1/docker-flow-proxy/config

docker service update \
    --mount-add "type=bind,source=$PWD/tmp/xip.io,target=/certs" \
    proxy

docker service inspect proxy --pretty


```