Docker Flow: Proxy
==================

The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide a simple way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and join them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) for service discovery. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Examples
--------

For a more detailed example, please read the [Docker Flow: Proxy - On-Demand HAProxy Reconfiguration](http://technologyconversations.com/2016/03/16/docker-flow-prox…-reconfiguration/) article.

Prerequisite for the *Docker Flow: Proxy* container is, at least, one [Consul](https://www.consul.io/) instance and the ability to put services information. The easiest way to store services information in Consul is through [Registrator]([Registrator](https://github.com/gliderlabs/registrator)).

To run the *Docker Flow: Proxy* container, please execute the following command (change *[CONSUL_IP]* with the address of the Consul instance).

```bash
docker run -d \
    --name docker-flow-proxy \
    --env CONSUL_ADDRESS=[CONSUL_IP]:8500 \
    -p 80:80
    docker-flow-proxy
```

The environment variable *CONSUL_ADDRESS* is mandatory.

Now you can deploy your services. Once a new service is running and its information is stored in Consul, run the `docker exec` command against the already running container.

```bash
docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

The `--service-name` must contain the name of the service that should be integrated into the proxy. That name must coincide with the name stored in Consul. The `--service-path` is the unique URL path that identifies the service. HAProxy will be configured to redirect all requests to the service starting with the value specified by the `--service-path` argument. Both of those arguments are mandatory.

In case more than one instance of the service is running, the proxy will load balance requests against all of them.

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
