# Docker Flow: Proxy - Swarm Mode (Docker 1.12+) With Automatic Configuration

* [Examples](#examples)

  * [Setup](#setup)
  * [Automatically Reconfiguring the Proxy](#automatically-reconfiguring-the-proxy)
  * [Removing a Service From the Proxy](#removing-a-service-from-the-proxy)
  * [Scaling the Proxy](#scaling-the-proxy)

* [The Flow Explained](#the-flow-explained)
* [Usage](../README.md#usage)

*Docker Flow: Proxy* running in the *Swarm Mode* is designed to leverage the features introduced in *Docker v1.12+*. If you are looking for a proxy solution that would work with older Docker versions or without Swarm Mode, please explore the [Docker Flow: Proxy - Standard Mode](standard-mode.md) article.

## Examples

The examples that follow assume that you have Docker Machine version v0.8+ that includes Docker Engine v1.12+. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*). Also, make sure that your Git client is configured to check out the code *AS-IS*. Otherwise, Windows might change carriage returns to the Windows format.

Please note that *Docker Flow: Proxy* is not limited to *Docker Machine*. We're using it as an easy way to create a cluster.

### Setup

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x scripts/swarm-cluster.sh

scripts/swarm-cluster.sh
```

Right now we have three machines running (*node-1*, *node-2*, and *node-3*). Each of those machines runs Docker Engine. Together, they form a Swarm cluster. Docker Engine running in the first node (*node-1*) is the leader.

We can see the cluster status by running the following command.

```bash
eval $(docker-machine env node-1)

docker node ls
```

We'll skip a detailed explanation of the Swarm cluster that is incorporated into Docker Engine 1.12. If you're new to it, please read [Docker Swarm Introduction](https://technologyconversations.com/2016/07/29/docker-swarm-introduction-tour-around-docker-1-12-series/). The rest of this article will assume that you have, at least, basic Docker 1.12+ knowledge.

Now we're ready to deploy a service.

### Automatically Reconfiguring the Proxy

We'll start by creating two networks.

```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo
```

The first network (*proxy*) will be dedicated to the proxy container and services that should be exposed through it. The second (*go-demo*) is the network used for communications between containers that constitute the *go-demo* service.

Next, we'll create the [swarm-listener](https://github.com/vfarcic/docker-flow-swarm-listener) service. It is companion to the `Docker Flow: Proxy`. Its purpose is to monitor Swarm services and send requests to the proxy whenever a service is created or destroyed.

Let's create the `swarm-listener` service.

> ## A note to Windows users
>
> For mounts to work, you will have to enter one of the machines before executing the `docker service create` command to work. To enter the Docker Machine, please execute the command that follows.
>
> `docker-machine ssh node-1`
>
> Please exit the machine once you finish executing the command that follows.
>
> `exit`

```bash
docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener
```

The service is attached to the proxy network, mounts the Docker socket, and declares the environment variables `DF_NOTIF_CREATE_SERVICE_URL` and `DF_NOTIF_REMOVE_SERVICE_URL`. We'll see the purpose of the variables soon. The service is constrained to the `manager` nodes.

The next step is to create the proxy service.

```bash
docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy
```

We opened the ports *80* and *443*. External requests will be routed through them towards destination services. The proxy is attached to the *proxy* network and has the mode set to *swarm*. The proxy must belong to the same network as the listener. They will exchange information whenever a service is created or removed.

Let's deploy the demo service. It consists of two containers; *mongo* is the database and *vfarcic/go-demo* is the actual service that uses it. They will communicate with each other through the *go-demo* network. Since we want to expose only *vfarcic/go-demo* to the "outside" world and keep the database "private", only the *vfarcic/go-demo* container will attach itself to the *proxy* network.

```bash
docker service create --name go-demo-db \
  --network go-demo \
  mongo
```

Let's run up the second service.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  vfarcic/go-demo
```

The details of the *go-demo* service are irrelevant for this exercise. What matters is that it was deployed somewhere inside the cluster and that it does not have any port exposed outside of the networks *go-demo* and *proxy*.

Please note the labels. They are a crucial part of the service definition. The `com.df.notify=true` tells the `Swarm Listener` whether to send a notifications whenever a service is created or removed. The rest of the labels match the query arguments we would use if we'd reconfigure the proxy manually. The only difference is that the labels are prefixed with `com.df`. For the list of the query arguments, please see the [Reconfigure](../README.md#reconfigure) section.

Now we should wait until all the services are running. You can see their status by executing the command that follows.

```bash
docker service ls
```

Once all the replicas are set to `1/1`, we can see the effect of the `com.df` labels by sending a request to the `go-demo` service through the proxy.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Thu, 13 Oct 2016 18:26:18 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

We sent a request to the proxy (the only service listening to the port 80) and got back the response from the `go-demo` service. The proxy was configured automatically as soon as the `go-demo` service was created.

The way the process works is as follows.

*Docker Flow: Swarm Listener* is running inside one of the Swarm manager nodes and queries Docker API in search for newly created services. Once it finds a new service, it looks for its labels. If the service contains the `com.df.notify` (it can hold any value), the rest of the labels with keys starting with `com.df.` are retrieved. All those labels are used to form request parameters. Those parameters are appended to the address specified as the `DF_NOTIF_CREATE_SERVICE_URL` environment variable defined in the `swarm-listener` service. Finally, a request is sent. In this particular case, the request was made to reconfigure the proxy with the service `go-demo` (the name of the service), using `/demo` as the path, and running on the port `8080`. The `distribute` label is not necessary in this example since we're running only a single instance of the proxy. However, in production we should run at least two proxy instances (for fault tolerance) and the `distribute` argument means that reconfiguration should be applied to all.

Please see the [Reconfigure](../README.md#reconfigure) section for the list of all the arguments that can be used with the proxy.

Since *Docker Flow: Proxy* uses new networking features added to Docker 1.12, it redirects all requests to the Swarm SDN (in this case called `proxy`). As a result, Docker takes care of load balancing, so there is no need to reconfigure it every time a new instance is deployed. We can confirm that by creating a few additional replicas.

```bash
docker service update --replicas 5 go-demo

curl -i $(docker-machine ip node-1)/demo/hello
```

Feel free to repeat this request a few more times. Once done, check the logs of any of the replicas and you'll notice that it received approximately one-fifth of the requests. No matter how many instances are running and with which frequency they change, Swarm networking will make sure that requests are load balanced across all currently running instances.

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example `servicePath` label attached to the `go-demo`service could be as follows.

```bash
...
  --label com.df.servicePath=/demo/hello,/demo/person \
...
```

Optionally, *serviceDomain* can be used as well. If specified, the proxy will allow access only to requests coming from that domain. The example label that follows would set *serviceDomain* to *my-domain.com*. After the proxy is reconfigured, only requests for that domain will be redirected to the destination service.

```bash
...
  --label com.df.serviceDomain=my-domain.com \
...
```

### Removing a Service From the Proxy

Since `Swarm Listener` is monitoring docker services, if a service is removed, related entries in the proxy configuration will be removed as well.

```bash
docker service rm go-demo
```

If you check the `Swarm Listener` logs, you'll see an entry similar to the one that follows.

```
Sending service removed notification to http://proxy:8080/v1/docker-flow-proxy/remove?serviceName=go-demo
```

A moment later, a new entry would appear in the proxy logs.

```
Processing request /v1/docker-flow-proxy/remove?serviceName=go-demo
Processing remove request /v1/docker-flow-proxy/remove
Removing go-demo configuration
Removing the go-demo configuration files
Reloading the proxy
```

From this moment on, the service *go-demo* is not available through the proxy.

`Swarm Listener` detected that the service was removed, send a notification to the proxy which, in turn, changed its configuration and reloaded underlying HAProxy.

Now that you've seen how to automatically add and remove services from the proxy, let's take a look at scaling options.

### Scaling the Proxy

Swarm is continuously monitoring containers health. If one of them fails, it will be redeployed to one of the nodes. If a whole node fails, Swarm will recreate all the containers that were running on that node. The ability to monitor containers health and make sure that they are (almost) always running is not enough. There is a brief period between the moment an instance fails until Swarm detects that and instantiates a new one. If we want to get close to zero-downtime systems, we must scale our services to at least two instances running on different nodes. That way, while we're waiting for one instance to recuperate from a failure, the others can take over its load. Even that is not enough. We need to make sure that the state of the failed instance is recuperated.

Let's see how *Docker Flow: Proxy* behaves when scaled.

Before we scale the proxy, we'll recreate the `go-demo` service that we removed a few moments ago.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  --replicas 3 \
  vfarcic/go-demo
```

At the moment we are still running a single instance of the proxy. Before we scale it, let's confirm that the listener sent a request to reconfigure it.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output should be status `200` indicating that the proxy works.

Let's scale the proxy to three instances.

```bash
docker service scale proxy=3
```

The proxy was scaled to three instances.

Normally, creating a new instance means that it starts without a state. As a result, the new instances would not have the `go-demo` service configured. Having different states among instances would produce quite a few undesirable effects. This is where the environment variable `LISTENER_ADDRESS` comes into play.

If you go back to the command we used to create the `proxy` service, you'll notice the argument that follows.

```bash
    -e LISTENER_ADDRESS=swarm-listener \
```

This tells the proxy the address of the `Docker Flow: Swarm Listener` service. Whenever a new instance of the proxy is created, it will send a request to the listener to resend notifications for all the services. As a result, each proxy instance will soon have the same state as the other.

If, for example, an instance of the proxy fails, Swarm will reschedule it and, soon afterwards, a new instance will be created. In that case, the process would be the same as when we scaled the proxy and, as the end result, the rescheduled instance will also have the same state as any other.

To test whether all the instances are indeed having the same configuration, we can send a couple of requests to the *go-demo* service.

Please run the command that follows a couple of times.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

Since Docker's networking (`routing mesh`) is performing load balancing, each of those requests is sent to a different proxy instance. Each was forwarded to the `go-demo` service endpoint, Docker networking did load balancing and resent it to one of the `go-demo` instances. As a result, all requests returned status *200 OK* proving that the combination of the proxy and the listener indeed works. All three instances of the proxy were reconfigured.

Before you start using `Docker Flow: Proxy`, you might want to get a better understanding of the flow of a request.

The Flow Explained
------------------

We'll go over the flow of a request to one of the services in the Swarm cluster.

A user or a service sends a request to our DNS (e.g. *acme.com*). The request is usually HTTP on the port `80` or HTTPS on the port `443`.

DNS resolves the domain to one of the servers inside the cluster. We do not need to register all the nodes. A few is enough (more than one in the case of a failure).

The Docker's routing mesh inspects which containers are running on a given port and re-sends the request to one of the instances of the proxy. It uses round robin load balancing so that all instances share the load (more or less) equally.

The proxy inspects the request path (e.g. `/demo/hello`) and sends it the end-point with the same name as the destination service (e.g. `go-demo`). Please note that for this to work, both the proxy and the destination service need to belong to the same network (e.g. `proxy`). The proxy changes the port to the one of the destination service (e.g. `8080`).

The proxy network performs load balancing among all the instances of the destination service, and re-sends the request to one of them.

The whole process sounds complicated (it actually is from the engineering point of view). But, as a user, all this is transparent.

One of the important things to note is that, with a system like this, everything can be fully dynamic. Before the new Swarm introduced in Docker 1.12, we would need to run our proxy instances on predefined nodes and make sure that they are registered as DNS records. With the new routing mesh, it does not matter whether the proxy runs on a node registered in DNS. It's enough to hit any of the servers, and the routing mesh will make sure that it reaches one of the proxy instances.

A similar logic is used for the destination services. The proxy does not need to do load balancing. Docker networking does that for us. The only thing it needs is the name of the service and that both belong to the same network. As a result, there is no need to reconfigure the proxy every time a new release is made or when a service is scaled.

Usage
-----

Please explore [Usage](../README.md#usage) for more information.
