Tour Around Docker 1.12: Docker Swarm
=====================================

Docker just published a new release v1.12 of the Engine. It is the most important release since v1.9. Back then, we got Docker networking that, finally, made it ready for use in clusters. With v1.12, Docker is reinventing itself with a completely new approach to cluster orchestration. Say good bye to Swarm as a separate container that depends on an external data store and please welcome *Docker service*. Everything you'll need to manage your cluster is now incorporated into Docker Engine. Swarm is there. Service discovery is there. Improved networking is there.

Since I believe that code (or in this case commands), explain things better then words, we'll start with a demo of some of the new features introduced in version 1.12. Specifically, we'll explore the new command *service*.

Environment Setup
-----------------

The examples that follow assume that you have [Docker Machine](https://www.docker.com/products/docker-machine) version v0.8+ that includes [Docker Engine](https://www.docker.com/products/docker-engine) v1.12+. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*).

We'll start by creating three machines that will simulate a cluster.

```
docker-machine create -d virtualbox node-1

docker-machine create -d virtualbox node-2

docker-machine create -d virtualbox node-3

docker-machine ls
```

The output of the `ls` command should be as follows.

```
NAME     ACTIVE   DRIVER       STATE     URL                         SWARM   DOCKER    ERRORS
node-1   -        virtualbox   Running   tcp://192.168.99.100:2376           v1.12.0
node-2   -        virtualbox   Running   tcp://192.168.99.101:2376           v1.12.0
node-3   -        virtualbox   Running   tcp://192.168.99.102:2376           v1.12.0
```

Please note that Docker version **MUST** be 1.12 or higher. If it isn't, please update your Docker Machine version, destroy the VMs, and start over.

With the machines up and running we can proceed and setup the Swarm cluster.

```bash
eval $(docker-machine env node-1)

docker swarm init --listen-addr $(docker-machine ip node-1):2377
```

The first command set environment variables so that local Docker Engine is pointing to the *node-1*. The second initialize Swarm on that machine. Right now, our Swarm cluster consists of only one VM.

Let's add the other two nodes to the cluster.

```bash
eval $(docker-machine env node-2)

docker swarm join $(docker-machine ip node-1):2377

eval $(docker-machine env node-3)

docker swarm join $(docker-machine ip node-1):2377
```

The other two machines joined the cluster as XXX. We can confirm that by sending `node ls` command to the *Leader* node (*node-1*).

```bash
eval $(docker-machine env node-1)

docker node ls
```

The output of the `node ls` command is as follows.

```
ID                           HOSTNAME  MEMBERSHIP  STATUS  AVAILABILITY  MANAGER STATUS
92ho364xtsdaq2u0189tna8oj *  node-1    Accepted    Ready   Active        Leader
c2tykql7a2zd8tj0b88geu45i    node-2    Accepted    Ready   Active
ejsjwyw5y92560179pk5drid4    node-3    Accepted    Ready   Active
```

The star tells us which node we can currently using. The *manager status* indicates that *node-1* is the *leader*.

In production environment we would probably set more than one node to be the leader and, thus, avoid deployment downtime if one of the fails. For the purpose of this demo, having one leader should suffice.

Deploying Container To The Cluster
----------------------------------



```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker network ls

docker service create --name go-demo-db \
  -p 27017 \
  --network go-demo \
  mongo

docker service ls # Repeat until go-demo-db REPLICAS is set to 1/1

docker service inspect -p go-demo-db

docker service tasks go-demo-db

docker service create --name go-demo \
  -p 8080 \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  vfarcic/go-demo

docker service ls # Repeat until go-demo REPLICAS is set to 1/1

docker service tasks go-demo

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    --constraint node.id==$(docker node inspect node-1 --format "{{.ID}}") \
    -e MODE=service \
    vfarcic/docker-flow-proxy

docker service ls # Repeat until proxy REPLICAS is set to 1/1

docker service tasks proxy

curl $(docker-machine ip node-1)/demo/hello

curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080"

curl $(docker-machine ip node-1)/demo/hello

docker service update --replicas 5 go-demo

docker service ls

docker service tasks go-demo

curl $(docker-machine ip node-1)/demo/hello
```

TODO
----

* Proxy

  * Note that sticky sessions will not work
  * Note that there is no need for Registrator

* Rolling updates (http://view.dckr.info:8080/#127)
* Failover
* Bundle
* SwarmKit (http://view.dckr.info:8080/#129)
* Logging (http://view.dckr.info:8080/#138)
