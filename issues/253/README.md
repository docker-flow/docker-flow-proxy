```bash
docker network create --driver overlay proxy

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener

docker service create \
       --name proxy \
       -p 8080:8080 -p 80:80 -p 443:443 \
       --network proxy \
       -e MODE=swarm \
       -e LISTENER_ADDRESS=swarm-listener \
       -e STATS_USER=haproxyuser \
       -e STATS_PASS=haproxypass \
       vfarcic/docker-flow-proxy:1.384

docker service create \
  --name pythonhttp \
  --network proxy \
  --mount type=bind,src=/path/to/files,dst=/var/www \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.serviceDomain=test.domain.com \
  --label com.df.servicePath=/ \
  --label com.df.port=8080 \
trinitronx/python-simplehttpserver
```