```bash
docker network create -d overlay proxy

docker stack deploy -c elk.yml elk

docker stack ps elk
```