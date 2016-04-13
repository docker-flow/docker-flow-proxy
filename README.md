Docker Flow: Proxy
==================

* [Introduction](#introduction)
* [Examples](#examples)
* [Containers Definition](#containers-definition)
* [Usage](#usage)
* [Feedback and Contribution](#feedback-and-contribution)

Introduction
------------

The goal of the *Docker Flow: Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) as service registry. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Prerequisite for the *Docker Flow: Proxy* container is, at least, one [Consul](https://www.consul.io/) instance and the ability to put services information. The easiest way to store services data in Consul is through [Registrator]([Registrator](https://github.com/gliderlabs/registrator)). That does not mean that Registrator is the requirement. Any other method that will put the information into Consul will do.

Examples
--------

For a more detailed example, please read the [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/) article. Besides providing more information, the article has a benefit or being OS agnostic. It will work on Linux, OS X, and Windows and do not have any requirement besides Vagrant.

The examples that follow assume that you have Docker Machine and Docker Compose installed. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox). The examples will not run on Windows. Please see the [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/) article for an OS agnostic walkthrough.

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x docker-flow-proxy-demo-environments.sh

./docker-flow-proxy-demo-environments.sh
```

Right now we have four machines running. The first one is called *proxy* and runs the containers *Consul* and *Docker Flow: Proxy*. The other three machines constitute the Docker Swarm cluster. Each of the machines in the cluster runs *Registrator* that monitors Docker events and puts data to Consul whenever a new container is run. It works the other way as well. If a container is stopped or removed, Registrator will eliminate its data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information regarding all containers running on the cluster.

Now we're ready to deploy a service.

```bash
eval "$(docker-machine env --swarm swarm-master)"

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    up -d
```

The details of the service we deployed are irrelevant for this exercise. What matters is that it was deployed somewhere inside the cluster and is running on a random port.

The only thing missing now is to reconfigure the proxy so that our newly deployed service is accessible on a standard HTTP port 80. That is the problem *Docker Flow: Proxy* is solving.

```bash
eval "$(docker-machine env proxy)"

export PROXY_IP=$(docker-machine ip proxy)

curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books"
```

That's it. All we had to do is send an HTTP request to `reconfigure` the proxy. The `serviceName` query contains the name of the service we want to integrate with the proxy. The `servicePath` is the unique URL that identifies the service.

The output of the `curl` command is as follows (formatted for better readability).

```json
{
  "Status": "OK",
  "Message": "",
  "ServiceName": "books-ms",
  "ServicePath": "/api/v1/books"
}
```

*Docker Flow: Proxy* responded saying that reconfiguration of the service *books-ms* running on the path */api/v1/books* was performed successfully.

Let's see whether the service is indeed accessible through the proxy.

```bash
curl -I $PROXY_IP/api/v1/books
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

The response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service.

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances.

```bash
eval "$(docker-machine env --swarm swarm-master)"

docker-compose \
    -f docker-compose-demo.yml \
    -p books-ms \
    scale app=3

curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books"

curl -I $PROXY_IP/api/v1/books
```

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example reconfiguration request could look as follows.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books,/api/v2/books" \
     | jq '.'
```

The result from the `curl` request is the reconfiguration of the *HAProxy* so that the *books-ms* service can be accessed through both the */api/v1/books* and the */api/v2/books* paths.

Optionally, *serviceDomain* can be used as well. If specified, the proxy will allow access only to requests coming from that domain. The example that follows sets *serviceDomain* to *my-domain-com*. After the proxy is reconfigured, only requests for that domain will be redirected to the destination service.

```bash
curl "$PROXY_IP:8080/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/&serviceDomain=my-domain.com" \
     | jq '.'
```

For a more detailed example, please read the [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/) article.

Containers Definition
---------------------

The complete definition of the containers we run can be found in the [docker-compose.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose.yml) file. The content is as follows.

```yml
version: '2'

services:
  consul:
    container_name: consul
    image: progrium/consul
    ports:
      - 8500:8500
      - 8301:8301
      - 8300:8300
    command: -server -bootstrap

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
    ports:
      - 80:80
      - 443:443
      - 8080:8080
```

Please note that this definition is compatible only with Docker Compose version 1.6+.

As you can see, all three targets are pretty simple and straightforward. In the examples, Consul was run on the *proxy* node. However, in production, you should probably run it on all servers in the cluster. Registrator was deployed to all Swarm nodes. Its `command` points to the *Consul* instance. Please note that *consul* and *registrator* targets are for demonstration purposes only. The only important requirement for the *proxy* target is the *CONSUL_ADDRESS* variable.

Finally, *proxy* target was also deployed to the *proxy* node. In production, you might want to run two instances of the *docker-flow-proxy* container and make sure that your DNS registries point to both of them. That way your traffic will not get affected in case one of those two nodes fail. The `CONSUL_ADDRESS` environment variable is mandatory and should contain the address of the Consul instance. Internal ports *80*, *443*, and *8080* can be exposed to any other port you prefer. HAProxy (inside the *docker-flow-proxy* container) is listening Ports *80* (HTTP) and *443* (HTTPS). The port *8080* is used to send *reconfigure* requests.

Usage
-----

### Reconfigure

> Reconfigures the proxy using information stored in Consul

The following query arguments can be used to send as a *reconfigure* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reconfigure**.

|Query      |Description                                                                 |Required|Example   |
|-----------|----------------------------------------------------------------------------|--------|----------|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |books-ms  |
|servicePath|The URL path of the service                                                 |Yes     |domain.com|
|pathType   |The ACL derivative. Defaults to *path_beg*. See [HAProxy path](https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path) for more info.|No|path_beg|

### Remove

> Removes a service from the proxy

The following query arguments can be used to send as a *remove* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/remove**.

|Query      |Description                                                                 |Required|Example   |
|-----------|----------------------------------------------------------------------------|--------|----------|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |books-ms  |

Feedback and Contribution
-------------------------

I'd appreciate any feedback you might give (both positive and negative). Feel fee to [create a new issue](https://github.com/vfarcic/docker-flow-proxy/issues), send a pull request, or tell me about any feature you might be missing. You can find my contact information in the [About](http://technologyconversations.com/about/) section of my [blog](http://technologyconversations.com/).
