```bash
docker-machine create -d virtualbox proxy

export DOCKER_IP=$(docker-machine ip proxy)

export CONSUL_IP=$(docker-machine ip proxy)

eval $(docker-machine env proxy)

docker-compose up -d consul-server registrator

docker-compose \
    -f docker-compose-demo2.yml \
    -p go-demo \
    up -d proxy db app

export PROXY_IP=$(docker-machine ip proxy)

curl -i $PROXY_IP/demo/hello

curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&consulTemplatePath=/test_configs/tmpl/go-demo.tmpl"

curl -i $PROXY_IP/demo/hello
```