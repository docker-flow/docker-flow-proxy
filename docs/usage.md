# Usage

*Docker Flow Proxy* can be reconfigured by sending HTTP requests or through Docker Service labels when combined with [Docker Flow Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener).

## Reconfigure

> Reconfigures the proxy

The proxy can be reconfigured to use request mode *http* or *tcp*. The default value of the request mode is *http* and can be changed with the parameter `reqMode`.

### Reconfigure General Parameters

The following query parameters can be used to send a *reconfigure* request to *Docker Flow Proxy*. They apply to any request mode and should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reconfigure**. The apply to any `reqMode`.

|Query          |Description                                                                               |Required|Default|Example      |
|---------------|------------------------------------------------------------------------------------------|--------|-------|-------------|
|aclName        |ACLs are ordered alphabetically by their names. If not specified, serviceName is used instead.|No  |       |05-go-demo-acl|
|httpsPort      |The internal HTTPS port of a service that should be reconfigured. The port is used only in the `swarm` mode. If not specified, the `port` parameter will be used instead.|No| |443|
|port           |The internal port of a service that should be reconfigured. The port is used only in the `swarm` mode. The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `port.1`, `port.2`, and so on).|Only in `swarm` mode| |8080|
|reqMode        |The request mode. The proxy should be able to work with any mode supported by HAProxy. However, actively supported and tested modes are `http`, `tcp`, and `sni`. The `sni` mode implies TCP with an SNI-based routing.|No|http|tcp|
|reqPathReplace |A regular expression to apply the modification. If specified, `reqPathSearch` needs to be set as well.|No| |/demo/|
|reqPathSearch  |A regular expression to search the content to be replaced. If specified, `reqPathReplace` needs to be set as well.|No| |/something/|
|serviceDomain  |The domain of the service. If set, the proxy will allow access only to requests coming to that domain. Multiple domains should be separated with comma (`,`).|No| |ecme.com|
|serviceDomainMatchAll|Whether to include subdomains and FDQN domains in the match. If set to false, and, for example, `serviceDomain` is set to `acme.com`, `something.acme.com` would not be considered a match unless this parameter is set to `true`. If this option is used, it is recommended to put any subdomains higher in the list using `aclName`.|No|false|true|
|serviceName    |The name of the service. It must match the name of the Swarm service or the one stored in Consul.|Yes|     |go-demo      |
|srcPort        |The source (entry) port of a service. Useful only when specifying multiple destinations of a single service. The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `srcPort.1`, `srcPort.2`, and so on).|No| |80|
|timeoutServer  |The server timeout in seconds.                                                            |No      |20     |60           |
|timeoutTunnel  |The tunnel timeout in seconds.                                                            |No      |3600   |1800         |
|xForwardedProto|Whether to add "X-Forwarded-Proto https" header.                                          |No      |false  |true         |

Multiple destinations for a single service can be specified by adding index as a suffix to `serviceDomain`, `srcPort`, and `port` parameters. In that case, `srcPort` is required. Defining multiple destinations is useful in cases when a service exposes multiple ports with different paths and functions.

### Reconfigure HTTP Mode Parameters

The following query parameters can be used only when `reqMode` is set to `http` or is empty.

|Query        |Description                                                                     |Required|Default|Example      |
|-------------|--------------------------------------------------------------------------------|--------|-------|-------------|
|consulTemplateBePath|The path to the Consul Template representing a snippet of the backend configuration. If set, proxy template will be loaded from the specified file.| | |/tmpl/be.tmpl|
|consulTemplateFePath|The path to the Consul Template representing a snippet of the frontend configuration. If set, proxy template will be loaded from the specified file.| | |/tmpl/fe.tmpl|
|distribute   |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|
|httpsOnly    |If set to true, HTTP requests to the service will be redirected to HTTPS.        |No      |false  |true         |
|outboundHostname|The hostname where the service is running, for instance on a separate swarm. If specified, the proxy will dispatch requests to that domain.|No| |ecme.com|
|pathType     |The ACL derivative. Defaults to *path_beg*. See [HAProxy path](https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path) for more info.|No| |path_beg|
|RedirectWhenHttpProto|Whether to redirect to https when X-Forwarded-Proto is set and the request is made over an HTTP port|No|false| |
|serviceCert  |Content of the PEM-encoded certificate to be used by the proxy when serving traffic over SSL.|No| | |
|servicePath  |The URL path of the service. Multiple values should be separated with comma (`,`). The parameter can be prefixed with an index thus allowing definition of multiple destinations for a single service (e.g. `servicePath.1`, `servicePath.2`, and so on).|Yes| |/api/v1/books|
|skipCheck    |Whether to skip adding proxy checks. This option is used only in the *default* mode.|No      |false  |true         |
|sslVerifyNone|If set to true, backend server certificates are not verified. This flag should be set for SSL enabled backend services.|No|false|true|
|templateBePath|The path to the template representing a snippet of the backend configuration. If specified, the backend template will be loaded from the specified file. If specified, `templateFePath` must be set as well. See the [Templates](#templates) section for more info.| | |/tmpl/be.tmpl|
|templateFePath|The path to the template representing a snippet of the frontend configuration. If specified, the frontend template will be loaded from the specified file. If specified, `templateBePath` must be set as well. See the [Templates](#templates) section for more info.| | |/tmpl/fe.tmpl|
|users        |A comma-separated list of credentials (<user>:<pass>) for HTTP basic authentication. It applies only to the service that will be reconfigured. If used with `usersSecret`, or when `USERS` environment variable is set, password may be omitted. In that case, it will be taken from `usersSecret` file or the global configuration if `usersSecret` is not present. |No| |usr1:pwd1, usr2:pwd2|
|usersSecret  |Suffix of Docker secret from which credentials will be taken for this service. Files must be a comma-separated list of credentials (<user>:<pass>). This suffix will be prepended with `dfp_users_`. For example, if the value is `mysecrets` the expected name of the Docker secret is `dfp_users_mysecrets`.|No| |monitoring|
|usersPassEncrypted|Indicates whether passwords provided by `users` or `usersSecret` contain encrypted data. Passwords can be encrypted with the command `mkpasswd -m sha-512 password1`|No|false|true|

Multiple destinations for a single service can be specified by adding index as a suffix to `servicePath`, `serviceDomain`, `srcPort`, and `port` parameters. In that case, `srcPort` is required. Defining multiple destinations is useful in cases when a service exposes multiple ports with different paths and functions.

### Reconfigure TCP Mode Parameters

The `reqMode` set to `tcp` does not have any specific parameters beyond those specified in the [Reconfigure General Parameters](#reconfigure-general-parameters) section.

!!! warning
    If multiple TCP services are defined to use the same `srcPort`, `serviceDomain` must be set for those services.

Please consult the [Using TCP Request Mode](swarm-mode-auto.md#using-tcp-request-mode) section for an example of working with `tcp` request mode.

An example request is as follows.

```
[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reconfigure?serviceName=foo&servicePath.1=/&port.1=8080&srcPort.1=80&servicePath.2=/&port.2=8081&srcPort.2=443
```

The command would create a service `foo` that exposes ports `8080` and `8081`. All requests coming to proxy port `80` with the path that starts with `/` will be forwarded to the service `foo` port `8080`. Equally, all requests coming to proxy port `443` (*HTTPS*) with the path that starts with `/` will be forwarded to the service `foo` port `8081`.

Indexes are incremental and start with `1`.

## Remove

> Removes a service from the proxy

The following query arguments can be used to send a *remove* request to *Docker Flow Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/remove**.

|Query      |Description                                                                 |Required|Default|Example|
|-----------|----------------------------------------------------------------------------|--------|-------|-------|
|aclName    |Mandatory if ACL name was specified in reconfigure request                  |No      |       |05-go-demo-acl|
|serviceName|The name of the service. It must match the name stored in Consul            |Yes     |       |go-demo|
|distribute |Whether to distribute a request to all the instances of the proxy. Used only in the *swarm* mode.|No|false|true|

## Certificates

All certificates stored in `/certs` directory are loaded automatically. If you already have a set of certificates you might choose to store them on a network drive and mount it to the service as `/certs`.

*Docker Flow Proxy* supports [Docker Secrets](https://docs.docker.com/engine/swarm/secrets/) for storing certificates. It will automatically load all certificate files added to the service as a secret. Only secrets with names that start with `cert-` or `cert_` will be considered a certificate.

During runtime, additional certificates can be added through [Put Certificate](#put-certificate) request.

Please consult [Configuring SSL Certificates](/certs) for a few examples of working with certificates.

## Put Certificate

> Puts SSL certificate to proxy configuration

The following query arguments can be used to send a *cert* request to *Docker Flow Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/cert**. Please note that the request method MUST be *PUT* and the certificate must be placed in request body.

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

!!! tip
    Use this feature only if your certificates are renewed often. To be on the safe side, it is recommended to mount `/certs` directory to a network drive and thus ensure that certs are preserved in case of a failure.

## Reload

> Reloads proxy configuration

The following query arguments can be used to send a *reload* request to *Docker Flow Proxy*. They should be added to the base address **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reload**.

|Query      |Description                                                |Required|Default|Example |
|-----------|-----------------------------------------------------------|--------|-------|--------|
|fromListener|Whether proxy configuration should be recreated from *Docker Flow Swarm Listener*. If set to true, configuration will be recreated independently of the `recreate` parameter. This operation is asynchronous.|No|false|true|
|recreate   |Recreates configuration using the information already available in the proxy. This param is useful in case config gets corrupted.|No|false|true|

An example is as follows.

```bash
curl -i \
    "[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/reload?recreate=false&fromListener=true"
```

## Config

> Outputs HAProxy configuration

The address is **[PROXY_IP]:[PROXY_PORT]/v1/docker-flow-proxy/config**

## Templates

Proxy configuration is a combination of configuration files generated from templates. Base template is `haproxy.tmpl`. Each service appends frontend and backend templates on top of the base template. Once all the templates are combined, they are converted into the `haproxy.cfg` configuration file.

The templates can be extended by creating a new Docker image based on `vfarcic/docker-flow-proxy` and adding the templates through `templateFePath` and `templateBePath` [reconfigure parameters](#reconfigure).

Templates are based on [Go HTML Templates](https://golang.org/pkg/html/template/).

Please see the [proxy/types.go](https://github.com/vfarcic/docker-flow-proxy/blob/master/proxy/types.go) for info about the structure used with templates.

