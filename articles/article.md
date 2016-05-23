Docker Flow: Proxy - On-Demand HAProxy Service Discovery and Reconfiguration (Docker Machine)
=============================================================================================

The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide a simple way to reconfigure proxy every time a new service is deployed or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) as service registry. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Before we jump into examples, let us discuss the need to use a dynamic proxy.

The Need for a Dynamic Proxy
----------------------------

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
---------------------------

The examples that follow assume that you have Docker Machine and Docker Compose installed. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*).

> I'm doing my best to make sure that all my articles with practical examples are working on a variety of OS and hardware combinations. However, making demos that always work is very hard, and you might experience some problems. In such a case, please contact me (my info is in the [About](http://technologyconversations.com/about/) section) and I'll do my best to help you out and, consequently, make the examples more robust.

Let's start by checking out the project code.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

To demonstrate the benefits of *Docker Flow: Proxy*, we'll set up a Swarm cluster and deploy a service. We'll create four virtual machines. One (*proxy*) will be running *Docker Flow: Proxy* and *Consul*. The other three machines will form the Swarm cluster with one master and two nodes.

Since I want to concentrate on how *Docker Flow: Proxy* works, I'll skip detailed instructions how will the environments be set up and only mention that we'll use the [docker-flow-proxy-demo-environments.sh script](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-flow-proxy-demo-environments.sh) that will create and provision the VMs with Docker Swarm, Consul, and Registrator.

Let us set up the cluster.

```
chmod +x docker-flow-proxy-demo-environments.sh

./docker-flow-proxy-demo-environments.sh
```

Creating and provisioning four servers requires a bit of time so grab some coffee and come back in a few minutes. We'll continue when the script is finished running.

Now we have the four servers (Docker Machines) up and running. The first one (*proxy*) is running [Consul](https://www.consul.io/) and *docker-flow-proxy* containers. Consul will contain all the information we might need for proxy configuration. At the same time, it is a service discovery tool of choice for setting up a Swarm cluster. In production, you should probably run it on all servers in the cluster but, for this demo, one instance should do.

The other three VMs constitute the cluster. Besides Swarm itself, each of the machines in the cluster is running *Registrator* that monitors Docker events and puts data into Consul whenever a new container is run. It works the other way as well. If a container is stopped or removed, Registrator will eliminate the data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information on all containers running on the cluster.

```bash
eval "$(docker-machine env --swarm swarm-master)"

docker info
```

As you can see, the Swarm cluster consists of three nodes (*swarm-master*, *swarm-node-1*, and *swarm-node-2*), each has one CPU and 1 GB of RAM, and their status is *Healthy*.

To summarize, we set up four servers. The *proxy* node hosts *Docker Flow: Proxy* and *Consul*. *Docker Flow: Proxy* will be our single entry into the system, and *Consul* will act as service registry. All requests to our services will go to a single port 80 in the *proxy* node, and HAProxy will make sure that they are redirected to the final destination.

The other three nodes constitute our Docker Swarm cluster. The *swarm-master* is in charge of orchestration and will deploy services to one of its nodes (*swarm-master*, *swarm-node-1*, and *swarm-node-2*). Each of those nodes is running *Registrator* that monitors Docker events and updates *Consul* information about deployed (or stopped) containers.

Now that everything is set up let's run a few services.

Reconfiguring Proxy With a Single Instance of a Service
-------------------------------------------------------

We'll run a service defined in the [docker-compose-demo2.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-demo2.yml) file.

```bash
docker-compose \
    -p go-demo \
    -f docker-compose-demo2.yml \
    up -d db app
```

We just run a service that exposes HTTP API. The details of the service are not important for this article. What matters is that it is running on a random port exposed by Docker Engine. Since we're using Swarm, it might be running on any of its nodes. In other words, both IP and port of the service are determined by Swarm instead being controlled by us. That is a good thing since Swarm takes away tedious task of managing services in a (potentially huge) cluster. However, not knowing IP and port in advance poses a few questions, most important one being how to access the service if we don't know where it is.

This is the moment when *Registrator* comes into play. It detected that a new container is running and stored its data into Consul. We can confirm that by running the following request.

```bash
export PROXY_IP=$(docker-machine ip proxy)

curl $PROXY_IP:8500/v1/catalog/service/go-demo?pretty
```

The output of the `curl` command is as follows.

```
[
    {
        "Node": "proxy",
        "Address": "192.168.99.100",
        "ServiceID": "bfdfeaf59e92:godemo_app_1:8080",
        "ServiceName": "go-demo",
        "ServiceTags": [],
        "ServiceAddress": "192.168.99.102",
        "ServicePort": 32768,
        "ServiceEnableTagOverride": false,
        "CreateIndex": 254,
        "ModifyIndex": 254
    }
]
```

We can see that, in this case, the service is running in *192.168.99.102* (*swarm-node-1*) on the port *32768*. The name of the service (*go-demo*) is the same as the name of the container we deployed. All we have to do now is reload the proxy.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo"
```

That's it. All we had to do is send an HTTP request to `reconfigure` the proxy. The `serviceName` query contains the name of the service we want to integrate with the proxy. It needs to match the *ServiceName* value stored in Consul. The `servicePath` is the unique URL that identifies the service. HAProxy will redirect all requests with URL that begin with that value.

The output of the `curl` command is as follows (formatted for readability).

```
{
  "Status": "OK",
  "Message": "",
  "ServiceName": "go-demo",
  "ServiceColor": "",
  "ServicePath": [
    "/demo"
  ],
  "ServiceDomain": "",
  "ConsulTemplatePath": "",
  "PathType": "",
  "SkipCheck": false
}
```

*Docker Flow: Proxy* responded saying that reconfiguration of the service *go-demo* running on the path */demo* was performed successfully.

Let's see whether the service is indeed accessible through the proxy.

```bash
curl -i $PROXY_IP/demo/hello
```

The output of the `curl` command is as follows.

```bash
HTTP/1.1 200 OK
Date: Sun, 22 May 2016 18:36:06 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

The response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service and its base URL.

Reconfiguring Proxy With a Multiple Instances of a Service
----------------------------------------------------------

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances of a service.

As an example, let's scale the service to three instances.

```bash
docker-compose \
    -f docker-compose-demo2.yml \
    -p go-demo \
    scale app=3
```

Let's see the result.

```bash
docker-compose \
    -f docker-compose-demo2.yml \
    -p go-demo \
    ps
```

The result of the `docker-compose ps` command is as follows.

```
    Name                  Command              State               Ports
-------------------------------------------------------------------------------------
godemo_app_1   /docker-entrypoint.sh go-demo   Up      192.168.99.102:32768->8080/tcp
godemo_app_2   /docker-entrypoint.sh go-demo   Up      192.168.99.102:32769->8080/tcp
godemo_app_3   /docker-entrypoint.sh go-demo   Up      192.168.99.101:32768->8080/tcp
godemo_db_1    /entrypoint.sh mongod           Up      27017/tcp
```

We can also confirm that Registrator picked up the events and stored the information to Consul.

```bash
curl $PROXY_IP:8500/v1/catalog/service/go-demo?pretty
```

This time, Consul returned different results.

```
[
    {
        "Node": "proxy",
        "Address": "192.168.99.100",
        "ServiceID": "9a9deaff202d:godemo_app_3:8080",
        "ServiceName": "go-demo",
        "ServiceTags": [],
        "ServiceAddress": "192.168.99.101",
        "ServicePort": 32768,
        "ServiceEnableTagOverride": false,
        "CreateIndex": 399,
        "ModifyIndex": 399
    },
    {
        "Node": "proxy",
        "Address": "192.168.99.100",
        "ServiceID": "bfdfeaf59e92:godemo_app_1:8080",
        "ServiceName": "go-demo",
        "ServiceTags": [],
        "ServiceAddress": "192.168.99.102",
        "ServicePort": 32768,
        "ServiceEnableTagOverride": false,
        "CreateIndex": 254,
        "ModifyIndex": 254
    },
    {
        "Node": "proxy",
        "Address": "192.168.99.100",
        "ServiceID": "bfdfeaf59e92:godemo_app_2:8080",
        "ServiceName": "go-demo",
        "ServiceTags": [],
        "ServiceAddress": "192.168.99.102",
        "ServicePort": 32769,
        "ServiceEnableTagOverride": false,
        "CreateIndex": 398,
        "ModifyIndex": 398
    }
]
```

As you can see, the service is scaled to three instances (not counting the database). One of them is running on *swarm-master* (*192.168.99.101*) while the other two were deployed to *swarm-node-1* (*192.168.99.102*). Even though three instances are running, the proxy continues redirecting all requests to the first instance. We can change that by re-running the `reconfigure` command.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo"
```

From this moment on, *HAProxy* is configured to perform load balancing across all three instances. We can continue scaling (and de-scaling) the service and, as long as we send the `reconfigure` request, the proxy will load-balance requests across all instances. They can be distributed among any number of servers, or even across different datacenters (as long as they are accessible from the proxy server).

Reconfiguring Proxy With Multiple Service Paths
-----------------------------------------------

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example reconfiguration request could look as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo/hello,/demo/person"
```

The result from the `curl` request is the reconfiguration of the *HAProxy* so that the *go-demo* service can be accessed through both the */demo/hello* and the */demo/person* base paths.

Reconfiguring Proxy Limited to a Specific Domain
------------------------------------------------

Optionally, serviceDomain can be used as well. If specified, the proxy will allow access only to requests coming from that domain. The example that follows sets *serviceDomain* to *my-domain-com*. After the proxy is reconfigured, only requests for that domain will be redirected to the destination service.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&serviceDomain=my-domain.com"
```

Removing a Service From the Proxy
---------------------------------

We can use *Docker Flow: Proxy* to also remove a service. An example that removes the service *go-demo* is as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/remove?serviceName=go-demo"
```

From this moment on, the service *go-demo* is not available through the proxy.

Reconfiguring the Proxy Using Custom Consul Templates
-----------------------------------------------------

In some cases, you might have a special need that requires a custom [Consul Template](https://github.com/hashicorp/consul-template). In such a case, you can expose the container volume and store your templates on the host. An example template can be found in the [test_configs/tmpl/go-demo.tmpl](https://github.com/vfarcic/docker-flow-proxy/tree/master/test_configs/tmpl/go-demo.tmpl) file. Its content is as follows.

```
frontend go-demo-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_go-demo path_beg /demo
	use_backend go-demo-be if url_go-demo

backend go-demo-be
	{{ range $i, $e := service "go-demo" "any" }}
	server {{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check
	{{end}}
```

This is a segment of an [HAProxy](http://www.haproxy.org/) configuration with a few Consul Template tags (those surrounded with `{{` and `}}`). Please consult HAProxy and Consul Template for more information.

This configuration file is available inside the container through a volume shared with the host. Please see the [Containers Definition](#containers-definition) for more info.

In this case, the path to the template residing inside the container is `/consul_templates/tmpl/go-demo.tmpl`. The request that would reconfigure the proxy using this template is as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&consulTemplatePath=/consul_templates/tmpl/go-demo.tmpl"
```

Proxy Failover
--------------

Consul is distributed service registry meant to run on multiple services (possible all servers in the cluster) and synchronize data across all instances. What that means is that as long as one Consul instance is available, data is available to whoever needs it.

Since *Docker Flow: Proxy* is designed to utilize information stored in Consul, it can recuperate its state from a failure without the need to persist data on disk.

Let's take a look at an example.

Before we simulate a failure, let's configure it (again) with the *go-demo* service we used throughout examples.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo/hello"

curl -i $PROXY_IP/demo/hello
```

The first command sent a reconfigure request and the second confirmed that the service is indeed available through the proxy.

We'll simulate a proxy failure by removing the container.

```bash
eval "$(docker-machine env proxy)"

docker-compose stop proxy

docker-compose rm -f proxy

curl -i $PROXY_IP/demo/hello
```

The output of the last command is as follows.

```bash
curl: (7) Failed to connect to 192.168.99.100 port 80: Connection refused
```

We proved that the proxy is indeed not running and that the *go-demo* service is not accessible.

We'll imagine that the whole node the proxy was running is down. In such a situation, we could try to fix the failing server or start the proxy somewhere else. In this example, we'll use the later option and start the proxy on *swarm-node-2*.

```bash
export CONSUL_IP=$(docker-machine ip proxy)

eval "$(docker-machine env swarm-node-2)"

docker-compose up -d proxy
```

The Docker Compose file we're using expects `CONSUL_IP` environment variable, so we set it up. Next, we run the proxy inside the *swarm-node-2* node.

Let's see whether our service is not accessible through the proxy.

```bash
export PROXY_IP=$(docker-machine ip swarm-node-2)

curl -i $PROXY_IP/demo/hello
```

The output of the request is as follows.

```
HTTP/1.1 200 OK
Date: Fri, 20 May 2016 12:07:34 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

On startup, *Docker Flow: Proxy* retrieved the information about all running services from Consul and recreated the configuration. It restored itself to the same state as it was before the failure.

Call For Action
---------------

Please give *Docker Flow: Proxy* a try. Deploy multiple services, scale them, destroy them, and so on. More information can be found in the project [README](https://github.com/vfarcic/docker-flow-proxy). Please contact me if you have a problem, suggestion, or an opinion regarding the project (my info is in the [About](http://technologyconversations.com/about/) section). Feel free to create a [New Issue](https://github.com/vfarcic/docker-flow-proxy/issues) or send a pull request.

Before moving onto a next task, please do not forget to stop (or destroy) VMs we created and free your resources.

```bash
docker-machine rm -f proxy swarm-master swarm-node-1 swarm-node-2
```

The DevOps 2.0 Toolkit
----------------------

<a href="https://leanpub.com/the-devops-2-toolkit" rel="attachment wp-att-3017"><img src="https://technologyconversations.files.wordpress.com/2014/04/the-devops-2-0-toolkit.png?w=188" alt="The DevOps 2.0 Toolkit" width="188" height="300" class="alignright size-medium wp-image-3017" /></a>If you liked this article, you might be interested in [The DevOps 2.0 Toolkit: Automating the Continuous Deployment Pipeline with Containerized Microservices](https://leanpub.com/the-devops-2-toolkit) book. Among many other subjects, it explores Docker, clustering, deployment, and scaling in much more detail.

The book is about different techniques that help us architect software in a better and more efficient way with *microservices* packed as *immutable containers*, *tested* and *deployed continuously* to servers that are *automatically provisioned* with *configuration management* tools. It's about fast, reliable and continuous deployments with *zero-downtime* and ability to *roll-back*. It's about *scaling* to any number of servers, the design of *self-healing systems* capable of recuperation from both hardware and software failures and about *centralized logging and monitoring* of the cluster.

In other words, this book envelops the whole *microservices development and deployment lifecycle* using some of the latest and greatest practices and tools. We'll use *Docker, Kubernetes, Ansible, Ubuntu, Docker Swarm and Docker Compose, Consul, etcd, Registrator, confd, Jenkins*, and so on. We'll go through many practices and, even more, tools.

The book is available from Amazon ([Amazon.com](http://www.amazon.com/dp/B01BJ4V66M) and other worldwide sites) and [LeanPub](https://leanpub.com/the-devops-2-toolkit).
