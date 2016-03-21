```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

vagrant plugin install vagrant-cachier

vagrant up swarm-master swarm-node-1 swarm-node-2 jenkins

vagrant ssh proxy

export DOCKER_HOST=tcp://10.100.192.200:2375

docker info

cd /vagrant

export DOCKER_HOST=tcp://swarm-master:2375

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    up -d

curl proxy:8500/v1/catalog/service/books-ms \
    | jq '.'

curl \
    "proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books" \
     | jq '.'

curl -I proxy/api/v1/books

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    scale app=3

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    ps

curl proxy:8500/v1/catalog/service/books-ms \
    | jq '.'

curl "proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books" \
     | jq '.'
exit

vagrant destroy -f
```
