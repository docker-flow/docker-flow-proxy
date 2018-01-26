```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker service create --name api-informes-db \
    --network go-demo \
    mongo:3.2.10


docker service create --name api-informes \
    -e DATABASE_URL=api-informes-db \
    -e DATABASE_PORT=27017 \
    -e DATABASE_NAME=sistac \
    -e APP_PORT=8080 \
    --network go-demo \
    --network proxy \
    jorgebo10/api-informes:1.3

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy

docker service ls

curl "http://localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=api-informes&servicePath=/&port=8080"

curl "http://localhost/api/informes/1" # working ok

curl "http://localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=api-informes&servicePath=/informes&port=8080"

curl "http://localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=api-informes&servicePath=/informes&port=8080&reqPathSearchReplace=/informes/,/"

curl "http://localhost/informes/api/informes/1" # not working anymore
```