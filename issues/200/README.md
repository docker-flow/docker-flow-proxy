```bash
docker network create -d overlay proxy

docker stack deploy -c proxy-stack.yaml proxy

docker network create -d overlay service

docker stack deploy -c services-stack.yaml services

docker stack deploy -c vote-stack.yaml vote

open "http://localhost/vote/"
```