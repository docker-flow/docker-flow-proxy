Docker Flow: Proxy
==================

* [Introduction](#introduction)
* [Modes](#modes)

  * [The Swarm Mode (Docker 1.12+) with automatic configuration](articles/swarm-mode-listener.md)
  * [The Swarm Mode (Docker 1.12+) with manual configuration](articles/swarm-mode.md)
  * [The Default Mode](articles/standard-mode.md)

* [Container Config](#container-config)

  * [Environment Variables](#environment-variables)
  * [Custom Config](#custom-config)
  * [Custom Errors](#custom-errors)

* [Usage](#usage)

  * [Reconfigure](#reconfigure)
  * [Remove](#remove)
  * [Put Certificate](#put-certificate)
  * [Reload](#reload)
  * [Config](#config)
  * [Templates](#templates)

* [Feedback and Contribution](#feedback-and-contribution)

## Introduction

The goal of the *Docker Flow: Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and adds custom logic that allows on-demand reconfiguration.

## Modes

Since the Docker 1.12 release, *Docker Flow: Proxy* supports two modes. The default mode is designed to work with any setup and requires Consul and Registrator. The **swarm** mode aims to leverage the benefits that come with *Docker Swarm* and new networking introduced in the 1.12 release. The later mode (*swarm*) does not have any dependency but Docker Engine. The *swarm* mode is recommended for all who use *Docker Swarm* features introduced in v1.12.

### [The Swarm Mode (Docker 1.12+) with automatic configuration](articles/swarm-mode-listener.md)
### [The Swarm Mode (Docker 1.12+) with manual configuration](articles/swarm-mode.md)
### [The Default Mode](articles/standard-mode.md)

## Container Config

### Environment Variables

> The *Docker Flow: Proxy* container can be configured through environment variables

The following environment variables can be used to configure the *Docker Flow: Proxy*.

|Variable           |Description                                               |Required|Default|Example|
|-------------------|----------------------------------------------------------|--------|-------|-------|
|CONSUL_ADDRESS     |The address of a Consul instance used for storing proxy information and discovering running nodes.  Multiple addresses can be separated with comma (e.g. 192.168.0.10:8500,192.168.0.11:8500).|Only in the *default* mode||192.168.0.10:8500|
|EXTRA_FRONTEND     |Value will be added to the default `frontend` configuration.|No    ||http-request set-header X-Forwarded-Proto https if { ssl_fc }|
|LISTENER_ADDRESS   |The address of the [Docker Flow: Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener) used for automatic proxy configuration.|Only in the *swarm* mode||swarm-listener|
|PROXY_INSTANCE_NAME|The name of the proxy instance. Useful if multiple proxies are running inside a cluster|No|docker-flow|docker-flow|
|MODE               |Two modes are supported. The *default* mode should be used for general purpose. It requires a Consul instance and service data to be stored in it (e.g. through Registrator). The *swarm* mode is designed to work with new features introduced in Docker 1.12 and assumes that containers are deployed as Docker services (new Swarm).|No      |default|swarm|
|SERVICE_NAME       |The name of the service. It must be the same as the value of the `--name` argument used to create the proxy service. Used only in the *swarm* mode.|No|proxy|my-proxy|
|STATS_USER         |Username for the statistics page                          |No      |admin  |my-user|
|STATS_PASS         |Password for the statistics page                          |No      |admin  |my-pass|
|TIMEOUT_CONNECT    |The connect timeout in seconds                            |No      |5      |3      |
|TIMEOUT_CLIENT     |The client timeout in seconds                             |No      |20     |5      |
|TIMEOUT_SERVER     |The server timeout in seconds                             |No      |20     |5      |
|TIMEOUT_QUEUE      |The queue timeout in seconds                              |No      |30     |10     |
|TIMEOUT_HTTP_REQUEST|The HTTP request timeout in seconds                      |No      |5      |3      |
|TIMEOUT_HTTP_KEEP_ALIVE|The HTTP keep alive timeout in seconds                |No      |15     |10     |
|USERS              |A comma-separated list of credentials(<user>:<pass>) for HTTP basic auth, which applies to all the backend routes.|No||user1:pass1,user2:pass2|

### Custom Config

The base HAProxy configuration can be found in [haproxy.tmpl](haproxy.tmpl). It can be customized by creating a new image. An example *Dockerfile* is as follows.

```
FROM vfarcic/docker-flow-proxy
COPY haproxy.tmpl /cfg/tmpl/haproxy.tmpl
```

### Custom Errors

Default error messages are stored in the `/errorfiles` directory inside the *Docker Flow: Proxy* image. They can be customized by creating a new image with custom error files or mounting a volume. Currently supported errors are `400`, `403`, `405`, `408`, `429`, `500`, `502`, `503`, and `504`.

## Usage

### Reconfigure

> Reconfigures the proxy using information stored in Consul

The following query arguments can be used to send as a *reconfigure* request to *Docker Flow: Proxy*. They should be added to the base address **<PROXY_IP>:<PROXY_PORT>/v1/docker-flow-proxy/reconfigure**.

|Query        |Description                                                                     |Required|Default|Example      |
|-------------|--------------------------------------------------------------------------------|--------|-------|-------------|
|aclName      |ACLs are ordered alphabetically by their names. If not specified, serviceName is used instead.|No||05-go-demo-acl|
|consulTemplateBePath|The path to the Consul Template representing a snippet of the backend configuration. If set, proxy template will be loaded from the specified file.|||/consul_templates/tmpl/go-demo-be.tmpl|
|consulTemplateFePath|The path to the Consul Template representing a snippet of the frontend configuration. If set, proxy template will be loaded from the specified file.|||/consul_templates/tmpl/go-demo-fe.tmpl|
|distribute   |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|
|httpsPort    |The internal HTTPS port of a service that should be reconfigured. The port is used only in the *swarm* mode. If not specified, the `port` parameter will be used instead.|No|||443|
|outboundHostname|The hostname where the service is running, for instance on a separate swarm. If specified, the proxy will dispatch requests to that domain.|No||machine123.internal.ecme.com|
|pathType     |The ACL derivative. Defaults to *path_beg*. See [HAProxy path](https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path) for more info.|No||path_beg|
|port         |The internal port of a service that should be reconfigured. The port is used only in the *swarm* mode. The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `port.1`, `port.2`, and so on).|Only in *swarm* mode||8080|
|reqPathReplace|A regular expression to apply the modification. If specified, `reqPathSearch` needs to be set as well.|No||/demo/|
|reqPathSearch |A regular expression to search the content to be replaced. If specified, `reqPathReplace` needs to be set as well.|No||/something/|
|serviceCert  |Content of the PEM-encoded certificate to be used by the proxy when serving traffic over SSL.|No|||
|serviceDomain|The domain of the service. If set, the proxy will allow access only to requests coming to that domain. Multiple domains should be separated with comma (`,`).|No||ecme.com|
|serviceName  |The name of the service. It must match the name of the Swarm service or the one stored in Consul.|Yes     |       |go-demo      |
|servicePath  |The URL path of the service. Multiple values should be separated with comma (`,`). The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `servicePath.1`, `servicePath.2`, and so on).|Yes||/api/v1/books|
|skipCheck    |Whether to skip adding proxy checks. This option is used only in the *default* mode.|No      |false  |true         |
|srcPort      |The source (entry) port of a service. Useful only when specifying multiple destinations of a single service. The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `srcPort.1`, `srcPort.2`, and so on).|No||80|
|templateBePath|The path to the template representing a snippet of the backend configuration. If specified, the backend template will be loaded from the specified file. If specified, `templateFePath` must be set as well. See the [Templates](#templates) section for more info.|||/templates/go-demo-be.tmpl|
|templateFePath|The path to the template representing a snippet of the frontend configuration. If specified, the frontend template will be loaded from the specified file. If specified, `templateBePath` must be set as well. See the [Templates](#templates) section for more info.|||/templates/go-demo-fe.tmpl|
|users        |A comma-separated list of credentials(<user>:<pass>) for HTTP basic auth, which applies only to the service that will be reconfigured.|No||user1:pass1,user2:pass2|

Multiple destinations for a single service can be specified by adding index as a suffix to `servicePath` and `port` parameters. In that case, `srcPort` is required. Defining multiple destinations is useful in cases when a service exposes multiple ports with different paths and functions.

An example request is as follows.

```
<PROXY_IP>:<PROXY_PORT>/v1/docker-flow-proxy/reconfigure?serviceName=foo&servicePath.1=/&port.1=8080&srcPort.1=80&servicePath.2=/&port.2=8081&srcPort.2=443
```

The command would create a service `foo` that exposes ports `8080` and `8081`. All requests coming to proxy port `80` with the path that starts with `/` will be forwarded to the service `foo` port `8080`. Equally, all requests coming to proxy port `443` (*HTTPS*) with the path that starts with `/` will be forwarded to the service `foo` port `8081`.

Indexes are incremental and start with `1`.

### Remove

> Removes a service from the proxy

The following query arguments can be used to send a *remove* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/remove**.

|Query      |Description                                                                 |Required|Default|Example|
|-----------|----------------------------------------------------------------------------|--------|-------|-------|
|aclName    |Mandatory if ACL name was specified in reconfigure request                  |No      |       |05-go-demo-acl|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |       |go-demo|
|distribute |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|

### Put Certificate

> Puts SSL certificate to proxy configuration

The following query arguments can be used to send a *cert* request to *Docker Flow: Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/cert**. Please note that the request method MUST be *PUT* and the certificate must be placed in request body.

When a new replica is deployed, it will synchronize with other replicas and recuperate their certificates.

|Query      |Description                                                                 |Required|Default|Example    |
|-----------|----------------------------------------------------------------------------|--------|-------|-----------|
|certName   |The file name of the certificate                                            |Yes     |       |my-cert.pem|
|distribute |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|

An example is as follows.

```bash
curl -i -XPUT \
    --data-binary @my-certificate.pem \
    "[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/cert?certName=my-certificate.pem&distribute=true"
```

Please note that the internal proxy port `8080` must be published.

The example would send a certificate stored in the `my-certificate.pem` file. The certificate would be distributed to all replicas of the proxy.

### Reload

> Reloads proxy configuration

The address is **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reload**

### Config

> Outputs HAProxy configuration

The address is **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/config**

### Templates

Proxy configuration is a combination of configuration files generated from templates. Base template is `haproxy.tmpl`. Each service appends frontend and backend templates on top of the base template. Once all the templates are combined, they are converted into the `haproxy.cfg` configuration file.

The templates can be extended by creating a new Docker image based on `vfarcic/docker-flow-proxy` and adding the templates through `templateFePath` and `templateBePath` [reconfigure parameters](#reconfigure).

Templates are based on [Go HTML Templates](https://golang.org/pkg/html/template/).

Please see the [actions/types.go](https://github.com/vfarcic/docker-flow-proxy/blob/master/actions/types.go) for info about the structure used with templates.

Feedback and Contribution
-------------------------

I'd appreciate any feedback you might give (both positive and negative). Feel fee to [create a new issue](https://github.com/vfarcic/docker-flow-proxy/issues), send a pull request, or tell me about any feature you might be missing. You can find my contact information in the [About](http://technologyconversations.com/about/) section of my [blog](http://technologyconversations.com/).

Please follow the [Contributing To The Project](articles/contribute.md) instructions before submitting a pull request.
