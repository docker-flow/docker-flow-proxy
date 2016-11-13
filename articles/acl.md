```bash
scripts/swarm-cluster.sh

scripts/swarm-services.sh

eval $(docker-machine env node-1)

docker service ls

curl -i "$(docker-machine ip node-1)/demo/hello" # 200

docker service update --publish-add 8080:8080 proxy

docker service ps proxy

curl -i "$(docker-machine ip node-1)/demo/hello" # 200

curl -i "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/remove?serviceName=go-demo&distribute=true"

curl -i "$(docker-machine ip node-1)/demo/hello" # 400

curl -i "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080&aclName=go-demo-acl&distribute=true"

curl -i "$(docker-machine ip node-1)/demo/hello" # 200

curl -i "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/remove?serviceName=go-demo&aclName=go-demo-acl&distribute=true"

curl -i "$(docker-machine ip node-1)/demo/hello" # 400
```