# Using Docker Stack To Run Docker Flow Proxy In Swarm Mode

This article assumes that you already understand how *Docker Flow Proxy* works. If you don't, please visit the [Running Docker Flow Proxy In Swarm Mode With Automatic Reconfiguration](swarm-mode-auto.md) page for a tutorial.

In this article, we'll explore how to create *Docker Flow Proxy* service through *Docker Compose* and the `docker stack deploy` command.

## Requirements

The examples that follow assume that you are using Docker v1.13+, Docker Compose v1.10+, and Docker Machine v0.9+.

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*). Also, make sure that your Git client is configured to check out the code *AS-IS*. Otherwise, Windows might change carriage returns to the Windows format.

Please note that *Docker Flow Proxy* is not limited to *Docker Machine*. We're using it as an easy way to create a cluster.

## Swarm Cluster Setup

Feel free to skip this section if you already have a working Swarm cluster.

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x scripts/swarm-cluster.sh

scripts/swarm-cluster.sh

eval $(docker-machine env node-1)
```

Now we're ready to deploy the `docker-flow-proxy` service.

## Running Docker Flow Proxy As A Docker Stack

We'll start by creating a network.

```bash
docker network create --driver overlay proxy
```

The *proxy* network will be dedicated to the proxy container and services that should be exposed through it.

We'll use [docker-compose-stack.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-stack.yml) from the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) repository to create `docker-flow-proxy` and `docker-flow-swarm-listener` services.

Content of the `docker-compose-stack.yml` file is as follows.

```
version: "3"

services:

  proxy:
    image: vfarcic/docker-flow-proxy
    ports:
      - 80:80
      - 443:443
    networks:
      - proxy
    environment:
      - LISTENER_ADDRESS=swarm-listener
      - MODE=swarm
    deploy:
      replicas: 2

  swarm-listener:
    image: vfarcic/docker-flow-swarm-listener
    networks:
      - proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure
      - DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove
    deploy:
      placement:
        constraints: [node.role == manager]

networks:
  proxy:
    external: true
```

The format is written in version 3 (mandatory for `docker stack deploy`).

It contains two services; `proxy` and `swarm-listener`. Since you are already familiar with *Docker Flow Proxy* and *Docker Flow Swarm Listener*, the arguments used with the services should be self explanatory.

The `proxy` network is defined as `external`. Even though `docker stack deploy` will create a default network for all the services that form the stack, the `proxy` network should be external so that we can attach services from other stacks to it.

TODO: Check whether the volume works on Windows.

Let's create the stack.

```bash
docker stack deploy -c docker-compose-stack.yml proxy
```

The first command created the services that form the stack defined in `docker-compose-stack.yml`.

The tasks of the stack can be seen through the `docker stack ps proxy` command.

```bash
docker stack ps proxy
```

The output is as follows (IDs are removed for brevity).

```
NAME                   IMAGE                                     NODE   DESIRED STATE CURRENT STATE         ERROR  PORTS
proxy_proxy.1          vfarcic/docker-flow-proxy:latest          node-2 Running       Running 2 minutes ago
proxy_swarm-listener.1 vfarcic/docker-flow-swarm-listener:latest node-1 Running       Running 2 minutes ago
proxy_proxy.2          vfarcic/docker-flow-proxy:latest          node-3 Running       Running 2 minutes ago
```

We are running two replicas of the `proxy` (in case of a failure) and one of the `swarm-listener`.

## Deploying Services Alongside Docker Flow Proxy

Let's deploy a demo stack. It consists of two containers; *mongo* is the database and *vfarcic/go-demo* is the actual service that uses it. They will communicate with each other through the *default* network of the stack. Since we want to expose only *vfarcic/go-demo* to the "outside" world and keep the database "private", only the *vfarcic/go-demo* container will attach itself to the *proxy* network.

TODO: Continue

```
# https://github.com/vfarcic/go-demo/blob/master/docker-compose-stack.yml

version: '3'

services:

  main:
    image: vfarcic/go-demo
    environment:
      - DB=db
    networks:
      - proxy
      - default
    deploy:
      replicas: 3
      labels:
        - com.df.notify=true
        - com.df.distribute=true
        - com.df.servicePath=/demo
        - com.df.port=8080

  db:
    image: mongo
    networks:
      - default

networks:
  default:
    external: false
  proxy:
    external: true
```

```bash
curl -o docker-compose-go-demo.yml \
    https://raw.githubusercontent.com/\
vfarcic/go-demo/master/docker-compose-stack.yml

docker stack deploy -c docker-compose-go-demo.yml go-demo

docker stack ps go-demo
```

```
ID            NAME                IMAGE                   NODE    DESIRED STATE  CURRENT STATE              ERROR                      PORTS
zqw94fk05c0m  go-demo_main.1      vfarcic/go-demo:latest  node-2  Running        Running 7 seconds ago
b6fgata0s3lx   \_ go-demo_main.1  vfarcic/go-demo:latest  node-3  Shutdown       Failed 22 seconds ago      "task: non-zero exit (2)"
d40gim4n7646   \_ go-demo_main.1  vfarcic/go-demo:latest  node-1  Shutdown       Failed 39 seconds ago      "task: non-zero exit (2)"
oqq15ztn0gi3   \_ go-demo_main.1  vfarcic/go-demo:latest  node-2  Shutdown       Failed 55 seconds ago      "task: non-zero exit (2)"
tqmwq05ydd86   \_ go-demo_main.1  vfarcic/go-demo:latest  node-3  Shutdown       Failed about a minute ago  "task: non-zero exit (2)"
mha3hpsgy81r  go-demo_db.1        mongo:latest            node-2  Running        Running 21 seconds ago
yvlwi44txmwh  go-demo_main.2      vfarcic/go-demo:latest  node-2  Running        Running 19 seconds ago
a9oby2nb0jks   \_ go-demo_main.2  vfarcic/go-demo:latest  node-3  Shutdown       Failed 35 seconds ago      "task: non-zero exit (2)"
4wway3he4cpg   \_ go-demo_main.2  vfarcic/go-demo:latest  node-1  Shutdown       Failed 51 seconds ago      "task: non-zero exit (2)"
fyfrhdq8a2hn   \_ go-demo_main.2  vfarcic/go-demo:latest  node-2  Shutdown       Failed about a minute ago  "task: non-zero exit (2)"
l33os992oea2   \_ go-demo_main.2  vfarcic/go-demo:latest  node-3  Shutdown       Failed about a minute ago  "task: non-zero exit (2)"
hw5vvfz2o639  go-demo_main.3      vfarcic/go-demo:latest  node-2  Running        Running 20 seconds ago
nmx651z6viuu   \_ go-demo_main.3  vfarcic/go-demo:latest  node-3  Shutdown       Failed 36 seconds ago      "task: non-zero exit (2)"
14az23d434l2   \_ go-demo_main.3  vfarcic/go-demo:latest  node-1  Shutdown       Failed 52 seconds ago      "task: non-zero exit (2)"
0x7ktxofhjct   \_ go-demo_main.3  vfarcic/go-demo:latest  node-2  Shutdown       Failed about a minute ago  "task: non-zero exit (2)"
zjoqix8fgt3y   \_ go-demo_main.3  vfarcic/go-demo:latest  node-3  Shutdown       Failed about a minute ago  "task: non-zero exit (2)"
```

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Thu, 19 Jan 2017 23:57:05 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

## Cleanup

Please remove Docker Machine VMs we created. You might need those resources for some other tasks.

```bash
docker-machine rm -f node-1 node-2 node-3
```
