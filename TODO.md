# TODO

* Write docs
* Test new config template
* Move logging server out of main

* haproxy.tmpl
* reconfigure.go
* reconfigure_test.go

```bash
docker-compose -f docker-compose-test.yml run --rm unit

docker build -t $DOCKER_HUB_USER/docker-flow-proxy:beta .

docker push $DOCKER_HUB_USER/docker-flow-proxy:beta

docker-compose -f docker-compose-test.yml run --rm staging-swarm
```