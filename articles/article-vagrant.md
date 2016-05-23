The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide a simple way to reconfigure proxy every time a new service is deployed or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) as service registry. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.
<!--more-->

Before we jump into examples, let us discuss the need to use a dynamic proxy.

The Need for a Dynamic Proxy
============================

I will assume that you are already using Docker to deploy your services and applications or, at least, that you know how it works. Among many benefits, Docker allows us to deploy immutable and easy to scale containers. When implementing containers with scaling in mind, one of the first things you should learn is that we should let Docker decide the port that will be exposed to the host. For example, if your service listens on the port 8080, that port should be defined internally inside the container but not exposed to a fixed number to the host. In other words, you should run the container with the flag `-p 8080` and not `-p 8080:8080`. The reason behind that is scaling. If the port exposed to the host is "hard-coded", we cannot scale that container due to potential conflicts. After all, two processes cannot listen to the same port. Even if you decide never to scale on the same server, hard-coding the port would mean that you need to keep a tight control over which ports are dedicated to each service. If you adopt microservices approach, the number of services will increase making ports management a nightmare. The second reason for not using pre-defined ports is proxy. It should be in charge or redirection of requests and load balancing ([HAProxy](http://www.haproxy.org/) and [nginx](http://nginx.org/) tend to be most common choices). Whenever a new container is deployed, the proxy needs to be reconfigured as well.

When you progress further with your Docker adoption, you will start using one of the cluster orchestration tools.
[Docker Swarm](https://www.docker.com/products/docker-swarm), for example, will make sure that containers are deployed somewhere within the cluster. It will choose the a server that best fits the needs. That greatly simplifies cluster management but poses a problem. If we do not know in advance on which server a container will be deployed, how will our users access the services? Not only that ports are unknown, but so are IPs. The solution is to reconfigure the proxy every time a new container is deployed or destroyed.

Successful management of a dynamic proxy reconfiguration lies in a couple of concepts and tools. We need a place to store the information of all running containers. That information should be reliable, fault-tolerant, and distributed. Some of the tools that fulfill those objectives are [Consul](https://www.consul.io/), [etcd](https://github.com/coreos/etcd), and [Zookeeper](https://zookeeper.apache.org/). For their comparison, please read the [Service Discovery: Zookeeper vs etcd vs Consul](http://technologyconversations.com/2015/09/08/service-discovery-zookeeper-vs-etcd-vs-consul/) article. Once we choose the tool we'll use to register our services, we need a way to update that data whenever new Docker events (`run`, `stop`, `rm`) are created. That can be easily accomplished with [Registrator](https://github.com/gliderlabs/registrator). It will update the service registry on every Docker event. Finally, with a service registry where we can store information and the way to monitor Docker events and update registry data, the only thing left is to update the proxy. How can we do that?

We can manually update the proxy configuration every time a new service is deployed. I guess there is no need to explain why this option should be discarded, so let's move on.

We can automatically update the proxy every time a new container is deployed. The solution could monitor Consul (or any other service registry) and apply some templating solution to update the proxy configuration. However, this is a very dangerous way to solve the problem since the proxy should not be updated immediately. Between deployment and proxy reconfiguration, we should run tests (let's call them post-deployment tests). They should confirm that the deployment was performed correctly, and there are no integration problems. Only if all post-deployment tests are successful, we should reconfigure the proxy and let our users benefit from new features. The best way to accomplish that is through the *blue-green deployment* process. Please see the [Docker Flow: Blue-Green Deployment and Relative Scaling](http://technologyconversations.com/tag/docker-flow/) for more information.

If both manual and automated *we-do-not-wait-for-anyone* proxy reconfigurations are discarded, the only solution left is on-demand reconfiguration. We should be able to deploy new containers to production in parallel with the old release, run post-deployment tests, and, in the end, tell the proxy to reconfigure itself. We should be able to have everything automated but decide ourselves when to trigger the change.

That is where [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) comes into play. It uses *HAProxy*, *Consul*, and adds an API that we can invoke whenever we want the configuration to change.

Let's see it in action.

Setting Up the Environments
===========================

> If you prefer using Docker Machine instead of Vagrant, please consult the example from the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project. If you have a problem, suggestion, or an opinion regarding the project, please send me an email (my info is in the [About](http://technologyconversations.com/about/) section) or create a [New Issue](https://github.com/vfarcic/docker-flow-proxy/issues)).

The only prerequisites for the example we are about to explore are [Vagrant](https://www.vagrantup.com/) and [Git client](https://git-scm.com/).

> I'm doing my best to make sure that all my articles with practical examples are working on a variety of OS and hardware combinations. However, making demos that always work is very hard, and you might experience some problems. In such a case, please contact me (my info is in the [About](http://technologyconversations.com/about/) section) and I'll do my best to help you out and, consequently, make the examples more robust.

Let's start by checking out the project code.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

To demonstrate the benefits of *Docker Flow: Proxy*, we'll set up a Swarm cluster and deploy a service. We'll create four virtual machines. One (*proxy*) will be running *Docker Flow: Proxy* and *Consul*. The other three machines will form the Swarm cluster with one master and two nodes.

Since I want to concentrate on how *Docker Flow: Proxy* works, I'll skip detailed instructions how will the environments be set up and only mention that we'll use [Ansible](https://www.ansible.com/) to provision the VMs.

Before we spin up the machines, you might want to install the *vagrant-cachier* plugin. It will cache dependencies so that subsequent package installations are done much faster.

```bash
vagrant plugin install vagrant-cachier
```

Let us create the VMs.

```
vagrant up swarm-master swarm-node-1 swarm-node-2 proxy
```

Creating and provisioning four servers requires a bit of time so grab some coffee and come back in a few minutes. We'll continue when `vagrant` command is finished.

Now we have the four servers up and running. The first one (*proxy*) is running [Consul](https://www.consul.io/) and *docker-flow-proxy* containers. Consul will contain all the information we might need for proxy configuration. At the same time, it is a service discovery tool of choice for setting up a Swarm cluster. In production, you should probably run it on all servers in the cluster but, for this demo, one instance should do.

The other three VMs constitute the cluster. Besides Swarm itself, each of the machines in the cluster is running *Registrator* that monitors Docker events and puts data into Consul whenever a new container is run. It works the other way as well. If a container is stopped or removed, Registrator will eliminate the data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information on all containers running on the cluster.

Let's enter the *proxy* VM and take a quick look at the cluster status.

```bash
vagrant ssh proxy

export DOCKER_HOST=tcp://10.100.192.200:2375

docker info
```

The output of the `docker info` is as follows.

```
Containers: 4
 Running: 4
 Paused: 0
 Stopped: 0
Images: 4
Server Version: swarm/1.1.3
Role: primary
Strategy: spread
Filters: health, port, dependency, affinity, constraint
Nodes: 2
 swarm-node-1: 10.100.192.201:2375
  └ Status: Healthy
  └ Containers: 2
  └ Reserved CPUs: 0 / 1
  └ Reserved Memory: 0 B / 1.536 GiB
  └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-79-generic, operatingsystem=Ubuntu 14.04.4 LTS, storagedriver=devicemapper
  └ Error: (none)
  └ UpdatedAt: 2016-03-21T14:24:54Z
 swarm-node-2: 10.100.192.202:2375
  └ Status: Healthy
  └ Containers: 2
  └ Reserved CPUs: 0 / 1
  └ Reserved Memory: 0 B / 1.536 GiB
  └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-79-generic, operatingsystem=Ubuntu 14.04.4 LTS, storagedriver=devicemapper
  └ Error: (none)
  └ UpdatedAt: 2016-03-21T14:24:51Z
Plugins:
 Volume:
 Network:
Kernel Version: 3.13.0-79-generic
Operating System: linux
Architecture: amd64
CPUs: 2
Total Memory: 3.072 GiB
Name: 9fdd284ff391
```

As you can see, the Swarm cluster consists of two nodes (*swarm-node-1* and *swarm-node-2*), each has one CPU and 1.5 GB of RAM, and the status is *Healthy*.

To summarize, we set up four servers. The *proxy* node hosts *Docker Flow: Proxy* and *Consul*. *Docker Flow: Proxy* will be our single entry into the system, and *Consul* will act as service registry. All requests to our services will go to a single port 80 in the *proxy* node, and HAProxy will make sure that they are redirected to the final destination.

The other three nodes constitute our Docker Swarm cluster. The *swarm-master* is in charge of orchestration and will deploy services to one of its nodes (at the moment only *swarm-node-1* and *swarm-node-2*). Each of those nodes is running *Registrator* that monitors Docker events and updates *Consul* information about deployed (or stopped) containers.

Now that everything is set up let's run a few services.

Reconfiguring Proxy With a Single Instance of a Service
=======================================================

We'll run a service defined in the [docker-compose-demo.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-demo.yml) file.

```bash
cd /vagrant

export DOCKER_HOST=tcp://swarm-master:2375

docker-compose
    -p books-ms
    -f docker-compose-demo.yml
    up -d
```

We just run a service that exposes HTTP API. The details of the service are not important for this article. What matters is that it is running on a random port exposed by Docker Engine. Since we're using Swarm, it might be running on any of its nodes. In other words, both IP and port of the service are determined by Swarm instead being controlled by us. That is a good thing since Swarm takes away tedious task of managing services in a (potentially huge) cluster. However, not knowing IP and port in advance poses a few questions, most important one being how to access the service if we don't know where it is.

This is the moment when *Registrator* comes into play. It detected that a new container is running and stored its data into Consul. We can confirm that by running the following request.

```bash
curl proxy:8500/v1/catalog/service/books-ms
    | jq '.'
```

The output of the `curl` command is as follows.

```
[
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.201",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-1:booksms_app_1:8080",
    "Address": "10.100.198.200",
    "Node": "proxy"
  }
]
```

We can see that, in this case, the service is running in *10.100.192.201* (*swarm-node-1*) on the port *32768*. The name of the service (*books-ms*) is the same as the name of the container we deployed. All we have to do now is reload the proxy.

```bash
curl
    "proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books"
     | jq '.'
```

That's it. All we had to do is send an HTTP request to `reconfigure` the proxy. The `serviceName` query contains the name of the service we want to integrate with the proxy. It needs to match the *ServiceName* value stored in Consul. The `servicePath` is the unique URL that identifies the service. HAProxy will redirect all requests with URL that begin with that value.

The output of the `curl` command is as follows.

```json
{
  "ServicePath": "/api/v1/books",
  "ServiceName": "books-ms",
  "Message": "",
  "Status": "OK"
}
```

*Docker Flow: Proxy* responded saying that reconfiguration of the service *books-ms* running on the path */api/v1/books* was performed successfully.

Let's see whether the service is indeed accessible through the proxy.

```bash
curl -I proxy/api/v1/books
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

The response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service and its base URL.

Reconfiguring Proxy With a Multiple Instances of a Service
==========================================================

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances of a service.

As an example, let's scale the service to three instances.

```bash
docker-compose
    -p books-ms
    -f docker-compose-demo.yml
    scale app=3
```

Let's see the result.

```bash
docker-compose
    -p books-ms
    -f docker-compose-demo.yml
    ps
```

The result of the `docker-compose ps` command is as follows.

```
    Name               Command          State               Ports
------------------------------------------------------------------------------
books-ms-db     /entrypoint.sh mongod   Up      27017/tcp
booksms_app_1   /run.sh                 Up      10.100.192.202:32768->8080/tcp
booksms_app_2   /run.sh                 Up      10.100.192.201:32768->8080/tcp
booksms_app_3   /run.sh                 Up      10.100.192.202:32769->8080/tcp
```

We can also confirm that Registrator picked up the events and stored the information to Consul.

```bash
curl proxy:8500/v1/catalog/service/books-ms
    | jq '.'
```

This time, Consul returned different results.

```
[
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.201",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-1:booksms_app_1:8080",
    "Address": "10.100.198.200",
    "Node": "proxy"
  },
  {
    "ServicePort": 32769,
    "ServiceAddress": "10.100.192.201",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-1:booksms_app_2:8080",
    "Address": "10.100.198.200",
    "Node": "proxy"
  },
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.202",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-2:booksms_app_3:8080",
    "Address": "10.100.198.200",
    "Node": "proxy"
  }
]
```

As you can see, the service is scaled to three instances (not counting the database). One of them are running on *swarm-node-1* (*10.100.192.201*) while the other two were deployed to *swarm-node-2* (*10.100.192.202*). Even though three instances are running, the proxy continues redirecting all requests to the first instance. We can change that by re-running the `reconfigure` command.

```bash
curl "proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books"
     | jq '.'
```

From this moment on, *HAProxy* is configured to perform load balancing across all three instances. We can continue scaling (and de-scaling) the service and, as long as we send the `reconfigure` request, the proxy will load-balance requests across all instances. They can be distributed among any number of servers, or even across different datacenters (as long as they are accessible from the proxy server).

Reconfiguring Proxy With Multiple Service Paths
===============================================

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example reconfiguration request could look as follows.

```bash
curl "proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books,/api/v2/books"
     | jq '.'
```

The result from the `curl` request is the reconfiguration of the *HAProxy* so that the *books-ms* service can be accessed through both the */api/v1/books* and the */api/v2/books* paths.

Call For Action
===============

Please give *Docker Flow: Proxy* a try. Deploy multiple services, scale them, destroy them, and so on. More information can be found in the project [README](https://github.com/vfarcic/docker-flow-proxy). Please contact me if you have a problem, suggestion, or an opinion regarding the project (my info is in the [About](http://technologyconversations.com/about/) section). Feel free to create a [New Issue](https://github.com/vfarcic/docker-flow-proxy/issues) or send a pull request.

Before moving onto a next task, please do not forget to stop (or destroy) VMs we created and free your resources.

```bash
exit

vagrant destroy -f
```
