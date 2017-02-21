```bash
docker network create --driver overlay proxy

curl -o docker-compose-stack.yml \
    https://raw.githubusercontent.com/\
vfarcic/docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c docker-compose-stack.yml proxy

curl -o docker-compose-go-demo.yml \
    https://raw.githubusercontent.com/\
vfarcic/go-demo/master/docker-compose-stack.yml

docker stack deploy -c docker-compose-go-demo.yml go-demo

docker stack ps go-demo

curl -i "localhost/demo/hello"

docker service update \
    --label-add com.df.serviceDomain=proxytest.local \
    go-demo_main

curl -i "http://proxytest.local./demo/hello"
```