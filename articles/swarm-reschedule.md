```bash
scripts/swarm.sh

eval "$(docker-machine env --swarm swarm-master)"

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    up -d

# docker-compose \
#     -p books-ms \
#     -f docker-compose-demo.yml \
#     scale app=2

docker ps --filter "name=bb1" --format "{{.Names"}}

docker-machine stop swarm-node-2

docker ps --filter "name=bb1" --format "{{.Names"}}

eval "$(docker-machine env swarm-master)"

docker logs swarm-agent-master # Removed Engine swarm-node-1

CONSUL_IP=$(docker-machine ip proxy)









eval "$(docker-machine env --swarm swarm-master)"

docker inspect b582c4d9e583364e8ba821731d73027c0f7904577b4f3305d001a8514e485863

docker-machine ssh swarm-master

```


```bash
# Add restart policy to registrator
./docker-flow-proxy-demo-environments.sh

eval "$(docker-machine env --swarm swarm-master)"

docker run -it alpine ping google.com

# TODO: Change to a container with a port
docker run -d -e reschedule:on-node-failure alpine ping google.com

docker-machine stop swarm-node-1 # Double check that's the node the container is running

docker info

docker logs -f swarm-agent-master
```

```
time="2016-06-01T11:36:26Z" level=info msg="Rescheduled container d755388cc4c22d37bdd63c21075a843a403a0935348f32269f5fa53e56908767 from swarm-node-1 to swarm-node-2 as f066f84aa1ea904cd927adfed72dd239957801378e20cd780e1ccb4b547d5d0b"
time="2016-06-01T11:36:26Z" level=info msg="Container d755388cc4c22d37bdd63c21075a843a403a0935348f32269f5fa53e56908767 was running, starting container f066f84aa1ea904cd927adfed72dd239957801378e20cd780e1ccb4b547d5d0b"
```

```bash
docker ps -a
```



```bash
./consul watch --http-addr 192.168.99.100:8500 -type services

./consul watch --http-addr 192.168.99.100:8500 -type service -service alpine tee consul.log
```