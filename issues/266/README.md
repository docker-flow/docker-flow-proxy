```bash
docker network create -d overlay proxy

docker stack deploy -c proxy.yml proxy

TAG=i-do-not-exist \
    docker stack deploy -c go-demo.yml go-demo
```