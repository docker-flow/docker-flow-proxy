```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy
```
