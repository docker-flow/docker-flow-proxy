```
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

docker stack ps -f desired-state=running proxy

curl -i "http://localhost/demo/hello"

docker service scale proxy_main=0

# docker service logs proxy_swarm-listener

# Repeat until we get response 503
curl -i "http://localhost/demo/hello"

docker service scale proxy_main=1

# docker service logs proxy_swarm-listener

# Repeat until we get response 200
curl -i "http://localhost/demo/hello"
```