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