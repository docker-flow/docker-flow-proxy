```bash
docker-machine create -d virtualbox \
  --virtualbox-memory "2500" \
  --virtualbox-disk-size "5000" node1

docker-machine create -d virtualbox \
  --virtualbox-memory "2500" \
  --virtualbox-disk-size "5000" node2

docker-machine ls

eval $(docker-machine env node1)

docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --name ucp docker/ucp install -i \
  --swarm-port 3376 --host-address $(docker-machine ip node1)

docker-machine rm -f node1 node2
```