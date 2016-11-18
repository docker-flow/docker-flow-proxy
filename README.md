Docker Flow: Proxy
==================

* [Introduction](#introduction)
* [Modes](#modes)

  * [The Swarm Mode (Docker 1.12+) with automatic configuration](articles/swarm-mode-listener.md)
  * [The Swarm Mode (Docker 1.12+) with manual configuration](articles/swarm-mode.md)
  * [The Default Mode](articles/standard-mode.md)

* [Containers Definition](#containers-definition)
* [Usage](#usage)

  * [Reconfigure](#reconfigure)
  * [Remove](#remove)
  * [Config](#config)

* [Feedback and Contribution](#feedback-and-contribution)

Introduction
------------

The goal of the *Docker Flow: Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and adds custom logic that allows on-demand reconfiguration.

Modes
-----

Since the Docker 1.12 release, *Docker Flow: Proxy* supports two modes. The default mode is designed to work with any setup and requires Consul and Registrator. The **swarm** mode aims to leverage the benefits that come with *Docker Swarm* and new networking introduced in the 1.12 release. The later mode (*swarm*) does not have any dependency but Docker Engine. The *swarm* mode is recommended for all who use *Docker Swarm* features introduced in v1.12.

### [The Swarm Mode (Docker 1.12+) with automatic configuration](articles/swarm-mode-listener.md)
### [The Swarm Mode (Docker 1.12+) with manual configuration](articles/swarm-mode.md)
### [The Default Mode](articles/standard-mode.md)

Usage
-----

### Container Config

> The *Docker Flow: Proxy* container can be configured through environment variables

The following environment variables can be used to configure the *Docker Flow: Proxy*.

|Variable           |Description                                               |Required|Default|Example|
|-------------------|----------------------------------------------------------|--------|-------|-------|
|CONSUL_ADDRESS     |The address of a Consul instance used for storing proxy information and discovering running nodes.  Multiple addresses can be separated with comma (e.g. 192.168.0.10:8500,192.168.0.11:8500).|Only in *default* mode||192.168.0.10:8500|
|LISTENER_ADDRESS   |The address of the [Docker Flow: Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener) used for automatic proxy configuration.|Only in *swarm* mode||swarm-listener|
|PROXY_INSTANCE_NAME|The name of the proxy instance. Useful if multiple proxies are running inside a cluster|No|docker-flow|docker-flow|
|MODE               |Two modes are supported. The *default* mode should be used for general purpose. It requires a Consul instance and service data to be stored in it (e.g. through Registrator). The *swarm* mode is designed to work with new features introduced in Docker 1.12 and assumes that containers are deployed as Docker services (new Swarm).|No      |default|swarm|
|SERVICE_NAME       |The name of the service. It must be the same as the value of the `--service` argument. Used only in the *swarm* mode.|No|proxy|my-proxy|

The base HAProxy configuration can be found in [haproxy.tmpl](haproxy.tmpl). It can be customized by creating a new container. An example *Dockerfile* is as follows.

```
FROM vfarcic/docker-flow-proxy
COPY haproxy.tmpl /cfg/haproxy.tmpl
```

### Reconfigure

> Reconfigures the proxy using information stored in Consul

The following query arguments can be used to send as a *reconfigure* request to *Docker Flow: Proxy*. They should be added to the base address **<PROXY_IP>:<PROXY_PORT>/v1/docker-flow-proxy/reconfigure**.

|Query        |Description                                                                     |Required|Default|Example      |
|-------------|--------------------------------------------------------------------------------|--------|-------|-------------|
|consulTemplateBePath|The path to the Consul Template representing a snippet of the backend configuration. If specified, the proxy template will be loaded from the specified file.|||/consul_templates/tmpl/go-demo-be.tmpl|
|consulTemplateFePath|The path to the Consul Template representing a snippet of the frontend configuration. If specified, the proxy template will be loaded from the specified file.|||/consul_templates/tmpl/go-demo-fe.tmpl|
|distribute   |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|
|pathType     |The ACL derivative. Defaults to *path_beg*. See [HAProxy path](https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path) for more info.|No||path_beg|
|port         |The internal port of a service that should be reconfigured. The port is used only in the *swarm* mode|Only in *swarm* mode|||8080|
|serviceDomain|The domain of the service. If specified, the proxy will allow access only to requests coming to that domain.|No||ecme.com|
|serviceName  |The name of the service. It must match the name stored in Consul.               |Yes     |       |books-ms     |
|servicePath  |The URL path of the service. Multiple values should be separated by a comma (,).|Yes (unless consulTemplatePath is present)||/api/v1/books|
|outboundHostname|The hostname where the service is running, for instance on a separate swarm. If specified, the proxy will dispatch requests to that domain.|No||machine123.internal.ecme.com|
|skipCheck    |Whether to skip adding proxy checks. This option is used only in the *default* mode.|No      |false  |true         |

### Remove

> Removes a service from the proxy

The following query arguments can be used to send a *remove* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/remove**.

|Query      |Description                                                                 |Required|Default|Example|
|-----------|----------------------------------------------------------------------------|--------|-------|-------|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |       |go-demo|
|distribute |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|

### Config

> Outputs HAProxy configuration

The address is **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/config**

Feedback and Contribution
-------------------------

I'd appreciate any feedback you might give (both positive and negative). Feel fee to [create a new issue](https://github.com/vfarcic/docker-flow-proxy/issues), send a pull request, or tell me about any feature you might be missing. You can find my contact information in the [About](http://technologyconversations.com/about/) section of my [blog](http://technologyconversations.com/).
