```bash
docker-machine create -d virtualbox \
  --virtualbox-memory "2500" \
  --virtualbox-disk-size "5000" node1

docker-machine ls

eval $(docker-machine env node1)

docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --name ucp docker/ucp install -i \
  --swarm-port 3376 --controller-port 8085 \
  --host-address $(docker-machine ip node1)

open "https://$(docker-machine ip node1):8085"

TOKEN=$(docker swarm join-token -q manager)

docker-machine create -d virtualbox \
  --virtualbox-memory "2500" \
  --virtualbox-disk-size "5000" node2

eval $(docker-machine env node2)

docker swarm join --token $TOKEN $(docker-machine ip node1):2377

docker-machine create -d virtualbox \
  --virtualbox-memory "2500" \
  --virtualbox-disk-size "5000" node3

eval $(docker-machine env node3)

docker swarm join --token $TOKEN $(docker-machine ip node1):2377

docker network create --attachable --driver overlay proxy

wget https://raw.githubusercontent.com/vfarcic/docker-flow-stacks\
/master/proxy/docker-flow-proxy-admin.yml

docker stack deploy -c docker-flow-proxy-admin.yml proxy

docker stack ps proxy # Wait until at least one proxy replica is running

docker network connect proxy ucp-controller

curl "http://$(docker-machine ip node1):8080/v1/docker-flow-proxy/reconfigure?serviceName=ucp-controller&servicePath=/&port=8080"

docker-machine rm -f node1 node2
```