Docker Flow: Proxy
==================

The goal of the *Docker Flow: Proxy* project is to provide a simple way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and join them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) for service discovery. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Prerequisite for the *Docker Flow: Proxy* container is, at least, one [Consul](https://www.consul.io/) instance and the ability to put services information. The easiest way to store services information in Consul is through [Registrator]([Registrator](https://github.com/gliderlabs/registrator)). That does not mean that Registrator is the requirement. Any other method that will put the information into Consul will do.

Examples
--------

For a more detailed example, please read the [Docker Flow: Proxy - On-Demand HAProxy Reconfiguration](http://technologyconversations.com/2016/03/16/docker-flow-proxy-reconfiguration/) article. Besides providing more information, the article has a benefit or being OS agnostic. It will work on Linux, OS X, and Windows and do not have any requirement besides Vagrant.

The example that follows assumes that you have Docker Machine and Docker Compose installed. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox). The examples will not run on Windows. Please see the [Docker Flow: Proxy - On-Demand HAProxy Reconfiguration](http://technologyconversations.com/2016/03/16/docker-flow-prox…-reconfiguration/) article for an OS agnostic walkthrough.

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x docker-flow-proxy-demo-environments.sh

./docker-flow-proxy-demo-environments.sh
```

Right now we have four machines running. The first one is called *proxy* and containers *Consul* and *Docker Flow: Proxy*. The other three machines form the Docker Swarm cluster. Each of the machines in the cluster have *Registrator* that monitors Docker events and puts data to Consul whenever a new container is run. It works in the other way as well. If a container is stopped or removed, Registrator will remove its data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information of all containers running inside the cluster.

Now we're ready to deploy a service.

```bash
eval "$(docker-machine env --swarm swarm-master)"

docker-compose \
    -p books-ms \
    -f docker-compose-demo.yml \
    up -d
```

The details of the service are irrelevant for this exercise. What matters is that it was deployed somewhere inside the cluster and is running on a random port.

The only thing missing now is to reconfigure the proxy so that our newly deployed service is accessible on a standard HTTP port 80. This is the problem *Docker Flow: Proxy* is solving.

```bash
eval "$(docker-machine env proxy)"

docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

That's it. All we had to do is run the `reconfigure` command together with a few arguments. The `--service-name` contains the name of the service we want to integrate with the proxy. The `--service-path` is the unique URL that identifies the service.

Let's see whether the service is indeed accessible through the proxy.

```bash
export PROXY_IP=$(docker-machine ip proxy)

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

For a more detailed example, please read the [Docker Flow: Proxy - On-Demand HAProxy Reconfiguration](http://technologyconversations.com/2016/03/16/docker-flow-proxy-reconfiguration/) article.

TODO
----

* Add description to Docker Hub
* Article

  * Proofread
  * Copy to README
  * Reference README, docker-flow README, and docker-flow article.
  * Publish

* New Docker Flow article with blue-green, scaling, and proxy

  * Write
  * Proofread
  * Publish
  * Add reference to README
