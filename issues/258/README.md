```bash
docker network create -d overlay proxy

docker stack deploy -c docker-compose-stack.yml dfp

docker stack deploy -c issues/258/stack.yml sa

open "http://localhost"

open "http://localhost:8081/counter"

open "http://localhost/counter"
```