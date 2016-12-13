```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x scripts/swarm-cluster.sh

scripts/swarm-cluster.sh

eval $(docker-machine env node-1)

docker node ls

docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker service create --name mysql_db \
    --network go-demo \
    shibli786/mysql

docker service create --name laravel_app \
    -e DB=mysql_db \
    --network go-demo \
    --network proxy \
    shibli786/laravelapp

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy

docker service ls # Wait until all replicas are running

curl "http://$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=laravel_app&servicePath=/&port=80"

curl "http://$(docker-machine ip node-1)/"

curl "http://$(docker-machine ip node-1):8080/v1/docker-flow-proxy/config"

docker service update --publish-add 1234:80 laravel_app

docker service ps laravel_app # Wait until it's updated

curl "http://$(docker-machine ip node-1):1234"
```