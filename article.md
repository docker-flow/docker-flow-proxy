Docker Flow: Dynamic Proxy
==========================

The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide an easy way to reconfigure proxy every time a new service is deployed or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and join them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) for service discovery. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Instead of debating theory, let's see it in action. We'll start by setting up an example environment.

Setting It Up
-------------

We'll use [Docker Compose](https://www.docker.com/products/docker-compose) and [Docker Machine](https://www.docker.com/products/docker-engine). If you do not already have them installed, the easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox). I'll assume that you already have a [Git client](https://git-scm.com/) set up.

Let's start by checking out the project code.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

To demonstrate the benefits of *Docker Flow: Proxy*, we'll setup a Swarm cluster and deploy a few services. We'll create four virtual machines. One (*proxy*), will be running *Consul* and *Docker Flow: Proxy*. The other three machines will form a Swarm cluster with one master and two nodes.

Let's start with the first one. We'll create the *proxy* machine and create a few environment variables

```
docker-machine create -d virtualbox proxy

export CONSUL_IP=$(docker-machine ip proxy)

export PROXY_IP=$(docker-machine ip proxy)
```

Now that the *proxy* VM is running, we can provision it with Consul and [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy). We'll

Now that we have a machine with Docker Engine up and running, we should create a few environment variables.

```
export DOCKER_IP=$(docker-machine ip docker-flow)

export CONSUL_IP=$(docker-machine ip proxy)
```

The first command was a standard *Docker Machine* way to enable a local Docker Engine to communicate with the one inside the VM we just created. The other two set the environment variables *DOCKER_IP* and *CONSUL_IP* that we'll use later on.

Now that all the prerequisites are set up, let's bring up the three containers that will constitute the solution for dynamic proxy. The [docker-compose.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose.yml) definition is as follows.

```yml
version: '2'

services:
  consul:
    container_name: consul
    image: progrium/consul
    ports:
      - 8500:8500
    command: -server -bootstrap

  registrator:
    depends_on:
      - consul
    container_name: registrator
    image: gliderlabs/registrator
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock
    command: -ip $DOCKER_IP consul://$CONSUL_IP:8500

  proxy:
    container_name: docker-flow-proxy
    image: vfarcic/docker-flow-proxy
    environment:
      CONSUL_ADDRESS: $CONSUL_IP:8500
    ports:
      - 80:80
    command: run
```

The configuration specifies three containers. Consul will be used as service registry and Registrator will make sure that Consul is updated whenever a new container is run or stopped. The third target is the custom made container called *docker-flow-proxy*. It is based on *HAProxy*, *Consul Template*, and custom code that will make sure that the proxy is reconfigured and points to all deployed services.

Let's run those containers.

```bash
eval "$(docker-machine env proxy)"

docker-compose up -d consul proxy
```

TODO: Start explain

```bash
eval "$(docker-machine env swarm-master)"

docker-compose up -d registrator swarm-master
```

TODO: End explain

Now that everything is set up, let's run a few services.

Running a Single Instance of a Service
----------------------------------------

We'll run services defined in the [docker-compose-demo.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-demo.yml).

```bash
docker-compose \
    -f docker-compose-demo.yml \
    up -d
```

We just run a service that exposes HTTP API. However, that service is running on a random port exposed by Docker Engine. This example is running on a single server. In production, we might choose to run containers in a cluster using Docker Swarm as orchestrator. In such a case, not only port, but also IP is not known in advance.

```bash
curl -I $DOCKER_IP/api/v1/books
```

The response of the `curl` command is as follows.

```
HTTP/1.0 503 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html
```

As expected, since HAProxy is not configured to responded with *503 Service Unavailable*. That means that we have to configure the proxy that will redirect all requests to the final service. *Docker Flow: Proxy* allows us to reconfigure the HAProxy instance with a single command.

```bash
docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

All we had to do is run the `reconfigure` command together with a few arguments. The `--service-name` contains the name of the service we want to integrate with the proxy. The `--service-path` is the unique URL that identifies the service.

Let's see whether it worked.

```bash
curl -I $DOCKER_IP/api/v1/books
```

The output of the `curl` command is as follows.

```bash
HTTP/1.1 200 OK
Server: spray-can/1.3.1
Date: Mon, 14 Mar 2016 22:08:11 GMT
Access-Control-Allow-Origin: *
Content-Type: application/json; charset=UTF-8
Content-Length: 2
```

This time, the response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service.

Scaling The Service
-------------------

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances.

As an example, let's scale the service to three instances.

```bash
docker-compose \
    -f docker-compose-demo.yml \
    scale app=3
```

Even though three instances are running, the proxy continues redirecting all requests to the first instances. We can change that by re-running the `reconfigure` command.

```bash
docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

From this moment on, HAProxy is reconfigured to perform load balancing across all three instances. We can continue scaling (and de-scaling) the service and, as long as the `reconfigure` command is run, the proxy will load balance all the requests. Those instances can be distributed among any number of servers, or even across different datacenters (as long as they are accessible from the proxy server).

Summary
-------

Even though the examples used a single service, you should not pose such a limit. You can use this project for as many services as you need. The only important prerequisite is that each has a unique name and a unique path.
