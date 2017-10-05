```
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

docker stack ps -f desired-state=running proxy

curl -i "http://localhost/demo/hello"

curl -k -i "https://localhost/demo/hello"
```