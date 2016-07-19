Integrating Proxy With Docker Swarm (Tour Around Docker 1.12 Series)
====================================================================

This article continues where [Introduction To The New Swarm](TODO) left. I will assume that you have at least basic knowledge how Swarm in Docker v1.12+ works. If you don't, please read the previous article first.

The fact that we can deploy any number of services inside a Swarm cluster does not mean that they are accessible to our users. We already saw that the new Swarm networking made it easy for services to communicate to each others. Let's explore how we can utilize it to expose them to the public. We'll try to integrate a proxy with the Swarm network and explore benefits version v1.12 brought.

Before we proceed, we need to setup a cluster we'll use for the examples.

Environment Setup
-----------------

The examples that follow assume that you have [Docker Machine](https://www.docker.com/products/docker-machine) version v0.8+ that includes [Docker Engine](https://www.docker.com/products/docker-engine) v1.12+. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*).

I won't go into details of the environment setup. It is the same as explained in the [Tour Around Docker 1.12: Docker Swarm](TODO) article. There will be three nodes, they will form a Swarm cluster.

> Please note that Docker version **MUST** be 1.12 or higher. If it isn't, please update your Docker Machine version, destroy the VMs, and start over.

```
docker-machine create -d virtualbox node-1

docker-machine create -d virtualbox node-2

docker-machine create -d virtualbox node-3

eval $(docker-machine env node-1)

docker swarm init \
    --secret my-secret \
    --auto-accept worker \
    --listen-addr $(docker-machine ip node-1):2377

eval $(docker-machine env node-2)

docker swarm join \
    --secret my-secret \
    $(docker-machine ip node-1):2377

eval $(docker-machine env node-3)

docker swarm join \
    --secret my-secret \
    $(docker-machine ip node-1):2377
```

Now that we have a Swarm cluster, we can deploy a service.

Deploying Services To The Cluster
---------------------------------

In order to utilize the new Docker Swarm networking, we'll start by creating two networks.

```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo
```

The first one (*proxy*) will be used for the communication between the proxy and the services that expose a public facing API. We'll use the second (*go-demo*) for all containers that form a service. In this case, we'll use the service *go-demo* that consists of two containers. It uses MongoDB to store data and *vfarcic/go-demo* as the back-end with an API.

We'll start with the database. Since it is not public-facing, there is no need to add it to the proxy. Therefore, we'll attach only to the *go-demo* network.

```bash
docker service create --name go-demo-db \
  -p 27017 \
  --network go-demo \
  mongo
```

With the database up and running, we can deploy the back-end. Since we want our external users to be able to use the API, we should integrate it with a proxy. Therefore, we should attach it to both networks (*proxy* and *go-demo*).

```bash
docker service create --name go-demo \
  -p 8080 \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  vfarcic/go-demo
```

Now both containers are running somewhere inside the cluster and are able to communicate with each other through the *go-demo* network. Let's bring a proxy into the mix. We'll use [HAProxy](http://www.haproxy.org/). The principles we'll explore are the same no matter which proxy you prefer.

Please note that we did not specify external ports. That means the neither is accessible except from containers that belong to the same networks.

Setting Up a Proxy Service
--------------------------

We can approach proxy in a couple of ways. One would be to create a new image based on *[HAProxy](https://hub.docker.com/_/haproxy/)* and include configuration files inside it. That approach would be good if the number of different services is relatively static. Otherwise, we'd need to create a new image with a new configuration every time there is a new service (not a new release). The second approach would be to expose a volume. That way, when needed, we could modify the configuration file instead building a whole new image. However, that has downsides as well. When deploying to a cluster, we should avoid using volumes whenever that's not necessary. As you'll soon see, proxy is one of those that do not require a volume. As a side note, `--volume` has been replaced with the `docker service` argument `--mount`.

The third option is to use one of the proxies designed to work with Docker. In this case, we'll use *[vfarcic/docker-flow-proxy](https://hub.docker.com/r/vfarcic/docker-flow-proxy/)* container, created from the *[Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy)* project. It is based on HAProxy with additional feature that allows use to reconfigure it by sending HTTP requests.

Let's give it a spin.

```bash
docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    --constraint node.hostname=="node-1" \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy
```

We opened ports *80* and *443* that will serve internet traffic (*HTTP* and *HTTPS*). The third port is *8080*. We'll use it to send requests to the proxy. Further on, we specified that it should belong to the *proxy* network. That way, since *go-demo* is also attached to the same network, the proxy can access it through the SDN.

Unlike other service that will be accessible through the proxy, we cannot allow this one to be deployed on a random node. That would pose difficulties when setting up the DNS. So, we want to be sure that it is running on a predetermined node. To accomplish that, we used the `--constraint` argument that specified the hostname of the destination server. Please note that the proxy is one of the very few services (if any) that should run on a specific server. We should let Swarm decide where services should be deployed. That does not mean that we shouldn't use constraints, but that they should be limited to resources we need. We should specify how many CPUs a service needs, how much memory, and so on. In other words, you should know where the proxy is in order to set up DNS records and let Swarm decide for all other services.

The last argument is the environment variable *MODE* that tells the proxy that containers will be deployed to a Swarm cluster. Please note that, in this context, Swarm needs to be based on Docker v1.12+. Please consult the project [README](https://github.com/vfarcic/docker-flow-proxy) for other combinations.

Before we proceed, let's confirm that proxy is running.

```bash
docker service tasks proxy
```

We can proceed if the *Last state* is *Running*. Otherwise, please wait until the service is up and running.

Now that the proxy is deployed, we should let it know about the existence of the *go-demo* service.

```bash
curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080"
```

The request was sent to *reconfigure* the proxy specifying the service name (*go-demo*), URL path of the API (*/demo*), and the internal port of the service (*8080*). From now on, all the requests to the proxy with the path that starts with */demo* will be redirected to the *go-demo* service.

We can test that the proxy indeed works as expected by sending an HTTP request.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output of the `curl` command is as follows.

```
HTTP/1.1 200 OK
Date: Mon, 18 Jul 2016 23:11:31 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

It responded with the HTTP status *200* and returned the API response *hello, world!*. The proxy works!

Let's explore the configuration generated by the proxy.

Proxy Configuration
-------------------

If you choose to roll-up your own proxy solution, it might be useful to understand how to configure the proxy and leverage new Docker networking features.

Let's start by examining the configuration *[Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy)* created for us. We can do that by entering the running container to take a sneak peek at the */cfg/haproxy.cfg* file. However, with Docker Swarm, finding a container is a bit tricky. If, for example, we deployed it with Docker Compose, the container name would be predictable. It would use <PROJECT>_<SERVICE>_<INDEX> format. The `docker service` command runs containers with hashed names. The *docker-flow-proxy* container created on my laptop has the name *proxy.1.e07jvhdb9e6s76mr9ol41u4sn*. Therefore, getting inside a running container deployed with Docker Swarm, we need to use a filter with, for example, image name.

Having all that into account, the command that will output configuration of the proxy is as follows.

```bash
docker exec -it \
    $(docker ps -q --filter "ancestor=vfarcic/docker-flow-proxy") \
    cat /cfg/haproxy.cfg
```

The important part of the configuration is as follows.

```
frontend services
    bind *:80
    bind *:443
    option http-server-close

    acl url_go-demo path_beg /demo
    use_backend go-demo-be if url_go-demo

backend go-demo-be
    server go-demo go-demo:8080
```

The first part (`frontend`) of presented configuration should be familiar to those who used HAProxy. It accepts requests on ports `80` (*HTTP*) and `443` (*HTTPS*). If path starts with `/demo`, it will be redirected to the `backend go-demo-be`. Inside it, requests are sent to the address `go-demo` on the port `8080`. The address is the service name. Since `go-demo` belongs to the same network as `proxy`, Docker will make sure that the request is redirected to the container. Neat, isn't it? There is no need, any more, to specify IPs and external ports.

The next question is how to do load balancing. How should we specify that the proxy should, for example, perform round-robin across all instances?

Load Balancing
--------------

Before we start load balancing explanation, let's create a few more instances of the *go-demo* service.

```bash
docker service update --replicas 5 go-demo
```

Within a few moments, five instances of the *go-demo* service will be running.

What should we do to make the proxy balance requests across all instances? The answer is nothing. No action is necessary from our part.

Normally, we would have something similar to the following configuration mock-up.

```
backend go-demo-be
    server instance_1 <INSTANCE_1_IP>:<INSTANCE_1_PORT>
    server instance_2 <INSTANCE_2_IP>:<INSTANCE_2_PORT>
    server instance_3 <INSTANCE_3_IP>:<INSTANCE_3_PORT>
    server instance_4 <INSTANCE_4_IP>:<INSTANCE_4_PORT>
    server instance_5 <INSTANCE_5_IP>:<INSTANCE_5_PORT>
```

However, with the new Docker networking inside a Swarm cluster, that is not necessary. It only introduces complications that require us to monitor instances and update the proxy every time a new replica is added or removed.

Docker will do load balancing for us. To be more precise, when the proxy redirects a request to `go-demo`, it is sent to Docker networking which, in turn, performs load balancing across all replicas (instances) of the service. The implication of this approach is that proxy is in charge of redirection from port *80* (or *443*) to the correct service inside the network.

Feel free to make requests to the service and inspect logs of one of the replicas. You'll see that, approximately, one fifth of the requests is sent to it.

Final Words
-----------

Docker networking introduced with the new Swarm included in Docker 1.12+ opens a door for quite a few new opportunities. Internal communication between containers and load balancing performed by Docker are only a few. Configuration of public facing proxies become easier than ever. We have make sure that all services that expose a public facing API are plugged into the same network as the proxy. From there on, all we have to do is configure the proxy to redirect all requests to a given service to its name. That will result in requests travelling from the proxy, to Docker network which, in turn, will perform load balancing across all instances.

The question that might arise is whether this approach is efficient. After all, we introduced a new layer. While, in the past we'd have only a proxy and a service, now we have Docker networking with load balancer in between. The answer is that overhead of such an approach is minimal. Docker uses [Linux IPVS](http://www.linuxvirtualserver.org/software/ipvs.html) for load balancing. It's been in the Linux kernel for more than fifteen years and proved to be one of the most efficient ways to load balance requests. Actually, it is much faster than *nginx* or *HAProxy*.

The next question is whether we need a proxy. We do. IPVS used by Docker will not do much more than than load balancing. We still need a proxy that will accept requests on ports *80* and *443* and, depending on their paths, redirect them to one service or another.

What are the downsides? The first one that comes to my mind are sticky sessions. If you expect the same user to send requests to the same instance, this approach will not work. A separate question is whether we should implement sticky sessions inside our services or as a separate entity. I'll leave that discussion for one of the next articles. Just keep in mind that sticky sessions will not work with this type of load balancing.

How about advantages? You already saw that simplicity is one of them. There's no need to reconfigure your proxy every time a new replica is deployed. As a result, the whole process is greatly simplified. Since we don't need the list of all IPs and ports of all instances, there is no need for tools like [Registrator](https://github.com/gliderlabs/registrator) and [Consul Template](https://github.com/hashicorp/consul-template). One of the possible flows in the past were to use Registrator to monitor Docker events and store IPs and ports in a key value store (e.g. [Consul](https://www.consul.io/)). Once information is stored, we would use Consul Template to recreate proxy configuration. There we many projects that simplified the process (one of them being the old version of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy)). However, with Docker Swarm and networking, the process just got simpler.

To *Docker Flow: Proxy* Or Not To *Docker Flow: Proxy*
------------------------------------------------------

I showed you how to configure HAProxy using [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project. It is HAProxy with an additional API that allows it to reconfigure the proxy with a simple HTTP request. It removes the need for manual configuration or templates.

On the other hand, rolling up your own proxy solution become easier then ever. With the few pointers from this article, you should have no problem to creating *nginx* or *HAProxy* configuration yourself.

My suggestion is to give [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) a try and make your own decision. In either case, new Docker Swarm features are really impressive and provide building blocks for more to come.

What Now?
---------

That concludes the exploration of some of the new Swarm and networking features we got with Docker v1.12. In particular, we explored those related to public facing proxies. Is this everything there is to know to run a Swarm cluster successfully? Not even close! What we explored by now (in this and the previous article) is only the beginning. There are quite a few questions to waiting to be answered. What happened to Docker Compose? How do we deploy new releases without downtime? Are there any additional tools we should use? I'll try to give answers to those and quite a few other questions in future articles. The next one will be dedicated to TODO.
