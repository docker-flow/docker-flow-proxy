```
docker network create -d overlay proxy

docker stack deploy -c proxy.yml proxy

docker stack ps proxy

docker service logs proxy_proxy

docker network create -d overlay ci

echo "admin" | docker secret create jenkins-user -

echo "admin" | docker secret create jenkins-pass -

docker stack deploy -c ci.yml ci

docker service logs proxy_proxy

open "http://localhost/jenkins"

docker network create -d overlay postgres

docker stack deploy -c db.yml db

docker stack ps db

docker service logs proxy_proxy
```