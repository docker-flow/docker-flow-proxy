```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

docker stack ps -f desired-state=running proxy

docker service logs proxy_proxy

ID=[...]

docker container exec -it $ID cat /cfg/haproxy.cfg
```