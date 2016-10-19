```bash
scripts/swarm-cluster.sh

eval $(docker-machine env node-1)

docker network create --driver overlay app1-net

docker network create --driver overlay proxy

docker service create --name proxy -p 80:80 -p 443:443 -p 8080:8080 --network proxy -e MODE=swarm vfarcic/docker-flow-proxy

docker service create --name go-demo-db --network proxy mongo

docker service create --name app1 --replicas 2 -e DB=go-demo-db --network app1-net --network proxy -e spring.profiles.active=prod vfarcic/go-demo

docker service create --name app2 --replicas 1 -e DB=go-demo-db --network proxy vfarcic/go-demo

docker service create --name app3 --replicas 2 -e DB=go-demo-db --network proxy vfarcic/go-demo

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=app1&servicePath=/&serviceDomain=app1.com&port=8080&distribute=true"

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=app2&servicePath=/&serviceDomain=app2.com&port=8080&distribute=true"

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=app3&servicePath=/&serviceDomain=swagger-editor.com&port=8080&distribute=true"

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/config"

for i in {1..10}; do
    curl -i app1.com/demo/person
done

# All 6 requests were sent to app1. Each instance received 3 requests

for i in 1 2 3 4 5 6; do
    curl -i app2.com/demo/person
done

# All 6 requests were sent to app3. Each instance received 3 requests

for i in 1 2 3 4 5 6; do
    curl -i swagger-editor.com/demo/person
done

# All 6 requests were sent to app3. Each instance received 3 requests
```

* Added mongo
* Changed to vfarcic/go-demo image
* serviceName app2r instead app2