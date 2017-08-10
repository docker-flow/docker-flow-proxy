```bash
docker network create -d overlay proxy

docker image build -t vfarcic/docker-flow-proxy:beta .

docker image push vfarcic/docker-flow-proxy:beta

TAG=beta docker stack deploy -c stack.yml proxy

curl "http://localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=xxx&servicePath=i-do-not-exist&port=1234"

curl "http://localhost:8080/v1/docker-flow-proxy/config?type=json"

curl "http://localhost:8080/v1/docker-flow-proxy/config"

docker stack rm proxy
```