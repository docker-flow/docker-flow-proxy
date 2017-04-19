# Production

## Deploy

```bash
docker network create --driver overlay proxy

curl -o proxy.yml \
    https://raw.githubusercontent.com/vfarcic/\
docker-flow-proxy/master/do/stack.yml

docker stack deploy -c proxy.yml proxy
```