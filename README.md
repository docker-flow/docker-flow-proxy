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

The goal of the *Docker Flow: Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and adds custom logic that allows on-demand reconfiguration.

Modes
-----

Since the Docker 1.12 release, *Docker Flow: Proxy* supports two modes. The default mode is designed to work with any setup and requires Consul and Registrator. The **service** mode is designed to leverage the benefits that come with *Docker Service* and new networking introduced in the 1.12 release. The later mode (*service*) does not have any dependency but Docker Engine. The *service* mode is recommended for all who use *Docker Service* features.

### [The Service Mode](articles/service-mode.md)
### [The Default Mode](articles/standard-mode.md)

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
|consulTemplateFePath|The path to the Consul Template representing snippet of the frontend configuration. If specified, proxy template will be loaded from the specified file.||/consul_templates/tmpl/go-demo-fe.tmpl|
|consulTemplateBePath|The path to the Consul Template representing snippet of the backend configuration. If specified, proxy template will be loaded from the specified file.||/consul_templates/tmpl/go-demo-be.tmpl|
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


TODO
----

* Add *MODE* env. variable
* Add *port* to the reconfigure query
* Node that skipCheck does not apply to the service mode