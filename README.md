Docker Flow: Proxy
==================

* [Introduction](#introduction)
* [Examples](#examples)
  * [Setup](#setup)
  * [Automatically Reconfiguring the Proxy](#automatically-reconfiguring-the-proxy)
  * [Removing a Service From the Proxy](#removing-a-service-from-the-proxy)
  * [Reconfiguring the Proxy Using Custom Consul Templates](#reconfiguring-the-proxy-using-custom-consul-templates)
  * [Proxy Failover](#proxy-failover)
* [Containers Definition](#containers-definition)
* [Usage](#usage)

  * [Reconfigure](#reconfigure)
  * [Remove](#remove)

* [Feedback and Contribution](#feedback-and-contribution)

Introduction
------------

The goal of the *Docker Flow: Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) as service registry. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Prerequisite for the *Docker Flow: Proxy* container is, at least, one [Consul](https://www.consul.io/) instance and the ability to put services information. The easiest way to store services data in Consul is through [Registrator]([Registrator](https://github.com/gliderlabs/registrator)). That does not mean that Registrator is the requirement. Any other method that will put the information into Consul will do.

Examples
--------

For a more detailed walkthrough and examples, please read the following articles:

* [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/)
* [Docker Networking and DNS: The Good, The Bad, And The Ugly](https://technologyconversations.com/2016/04/25/docker-networking-and-dns-the-good-the-bad-and-the-ugly/)

The examples that follow assume that you have Docker Machine and Docker Compose installed. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*).

### Setup

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x docker-flow-proxy-demo-environments.sh

./docker-flow-proxy-demo-environments.sh
```

Right now we have four machines running. The first one is called *proxy* and runs the containers *Consul* and *Docker Flow: Proxy*. The other three machines constitute the Docker Swarm cluster. Each of the machines in the cluster runs *Registrator* that monitors Docker events and puts data to Consul whenever a new container is run. It works the other way as well. If a container is stopped or removed, Registrator will eliminate its data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information regarding all containers running on the cluster.

Now we're ready to deploy a service.

### Automatically Reconfiguring the Proxy

```bash
eval "$(docker-machine env --swarm swarm-master)"

docker-compose \
    -p go-demo \
    -f docker-compose-demo2.yml \
    up -d db app
```

The details of the service we deployed are irrelevant for this exercise. What matters is that it was deployed somewhere inside the cluster and is running on a random port.

The only thing missing now is to reconfigure the proxy so that our newly deployed service is accessible on a standard HTTP port 80. That is the problem *Docker Flow: Proxy* is solving.

```bash
export PROXY_IP=$(docker-machine ip proxy)

curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo"
```

That's it. All we had to do is send an HTTP request to `reconfigure` the proxy. The `serviceName` query contains the name of the service we want to integrate with the proxy. The `servicePath` is the unique URL that identifies the service.

The output of the `curl` command is as follows (formatted for better readability).

```json
{
  "Status": "OK",
  "Message": "",
  "ServiceName": "go-demo",
  "ServiceColor": "",
  "ServicePath": [
    "/demo"
  ],
  "ServiceDomain": "",
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
Date: Thu, 19 May 2016 19:21:55 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

The response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service.

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances.

```bash
docker-compose \
    -f docker-compose-demo2.yml \
    -p go-demo \
    scale app=3

curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo/hello"

curl -i $PROXY_IP/demo/hello
```

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example reconfiguration request could look as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo/hello,/demo/person"
```

The result from the `curl` request is the reconfiguration of the *HAProxy* so that the *go-demo* service can be accessed through both the */demo/hello* and the */demo/person* paths.

Optionally, *serviceDomain* can be used as well. If specified, the proxy will allow access only to requests coming from that domain. The example that follows sets *serviceDomain* to *my-domain-com*. After the proxy is reconfigured, only requests for that domain will be redirected to the destination service.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&serviceDomain=my-domain.com"
```

For a more detailed example, please read the [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/) article.

### Removing a Service From the Proxy

We can use *Docker Flow: Proxy* to also remove a service. An example that removes the service *go-demo* is as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/remove?serviceName=go-demo"
```

From this moment on, the service *go-demo* is not available through the proxy.

### Reconfiguring the Proxy Using Custom Consul Templates

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

### Proxy Failover

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

Containers Definition
---------------------

The complete definition of the containers we run can be found in the [docker-compose.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose.yml) file. The content is as follows.

```yml
version: '2'

services:

  consul-server:
    container_name: consul
    image: consul
    network_mode: host
    environment:
      - 'CONSUL_LOCAL_CONFIG={"skip_leave_on_interrupt": true}'
    command: agent -server -bind=$DOCKER_IP -bootstrap-expect=1 -client=$DOCKER_IP

  registrator:
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
    volumes:
      - ./test_configs/:/consul_templates/
    ports:
      - 80:80
      - 443:443
      - 8080:8080
```

Please note that this definition is compatible only with Docker Compose version 1.6+.

As you can see, all three targets are pretty simple and straightforward. In the examples, Consul was run on the *proxy* node. However, in production, you should probably run it on all servers in the cluster. Registrator was deployed to all Swarm nodes. Its `command` points to the *Consul* instance. Please note that *consul* and *registrator* targets are for demonstration purposes only. The only important requirement for the *proxy* target is the *CONSUL_ADDRESS* variable.

Finally, the *proxy* target was also deployed to the *proxy* node. In production, you might want to run two instances of the *docker-flow-proxy* container and make sure that your DNS registries point to both of them. That way your traffic will not get affected in case one of those two nodes fail. The `CONSUL_ADDRESS` environment variable is mandatory and should contain the address of the Consul instance. Internal ports *80*, *443*, and *8080* can be exposed to any other port you prefer. HAProxy (inside the *docker-flow-proxy* container) is listening Ports *80* (HTTP) and *443* (HTTPS). The port *8080* is used to send *reconfigure* requests.

Usage
-----

### Reconfigure

> Reconfigures the proxy using information stored in Consul

The following query arguments can be used to send as a *reconfigure* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reconfigure**.

|Query        |Description                                                                     |Required|Default|Example      |
|-------------|--------------------------------------------------------------------------------|--------|-------|-------------|
|serviceName  |The name of the service. It must match the name stored in Consul.               |Yes     |       |books-ms     |
|servicePath  |The URL path of the service. Multiple values should be separated with comma (,).|Yes (unless consulTemplatePath is present)||/api/v1/books|
|serviceDomain|The domain of the service. If specified, proxy will allow access only to requests coming to that domain.|No||ecme.com|
|pathType     |The ACL derivative. Defaults to *path_beg*. See [HAProxy path](https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path) for more info.|No||path_beg|
|consulTemplatePath|The path to the Consul Template. If specified, proxy template will be loaded from the specified file.|Yes (unless servicePath is present)||/consul_templates/tmpl/go-demo.tmpl|
|skipCheck    |Whether to skip adding proxy checks.                                            |No      |false  |true         |

### Remove

> Removes a service from the proxy

The following query arguments can be used to send as a *remove* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/remove**.

|Query      |Description                                                                 |Required|Example   |
|-----------|----------------------------------------------------------------------------|--------|----------|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |books-ms  |

Feedback and Contribution
-------------------------

I'd appreciate any feedback you might give (both positive and negative). Feel fee to [create a new issue](https://github.com/vfarcic/docker-flow-proxy/issues), send a pull request, or tell me about any feature you might be missing. You can find my contact information in the [About](http://technologyconversations.com/about/) section of my [blog](http://technologyconversations.com/).
