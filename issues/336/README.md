```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

docker stack ps proxy

docker container ls

ID=28d9f4735a8d

docker container exec -it $ID sh

ps aux

pkill haproxy
```