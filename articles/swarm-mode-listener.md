# Docker Flow: Proxy - Swarm Mode (Docker 1.12+) With Automatic Configuration

* [Examples](#examples)

  * [Setup](#setup)
  * [Automatically Reconfiguring the Proxy](#automatically-reconfiguring-the-proxy)
  * [Removing a Service From the Proxy](#removing-a-service-from-the-proxy)
  * [Scaling the Proxy](#scaling-the-proxy)
  * [Rewriting Paths](#rewriting-paths)
  * [Basic Authentication](#basic-authentication)

    * [Global Authentication](#global-authentication)
    * [Service Authentication](#service-authentication)

  * [Configuring Service SSLs And Proxying HTTPS Requests](#configuring-service-ssls-and-proxying-https-requests)

* [The Flow Explained](#the-flow-explained)
* [Usage](../README.md#usage)

*Docker Flow: Proxy* running in the *Swarm Mode* is designed to leverage the features introduced in *Docker v1.12+*. If you are looking for a proxy solution that would work with older Docker versions or without Swarm Mode, please explore the [Docker Flow: Proxy - Standard Mode](standard-mode.md) article.

## Examples

The examples that follow assume that you have Docker Machine version v0.8+ that includes Docker Engine v1.12+. The easiest way to get them is through [Docker Toolbox](https://www.docker.com/products/docker-toolbox).

> If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*). Also, make sure that your Git client is configured to check out the code *AS-IS*. Otherwise, Windows might change carriage returns to the Windows format.

Please note that *Docker Flow: Proxy* is not limited to *Docker Machine*. We're using it as an easy way to create a cluster.

### Setup

To setup an example environment using Docker Machine, please run the commands that follow.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy

chmod +x scripts/swarm-cluster.sh

scripts/swarm-cluster.sh
```

Right now we have three machines running (*node-1*, *node-2*, and *node-3*). Each of those machines runs Docker Engine. Together, they form a Swarm cluster. Docker Engine running in the first node (*node-1*) is the leader.

We can see the cluster status by running the following command.

```bash
eval $(docker-machine env node-1)

docker node ls
```

We'll skip a detailed explanation of the Swarm cluster that is incorporated into Docker Engine 1.12. If you're new to it, please read [Docker Swarm Introduction](https://technologyconversations.com/2016/07/29/docker-swarm-introduction-tour-around-docker-1-12-series/). The rest of this article will assume that you have, at least, basic Docker 1.12+ knowledge.

Now we're ready to deploy a service.

### Automatically Reconfiguring the Proxy

We'll start by creating two networks.

```bash
docker network create --driver overlay proxy

docker network create --driver overlay go-demo
```

The first network (*proxy*) will be dedicated to the proxy container and services that should be exposed through it. The second (*go-demo*) is the network used for communications between containers that constitute the *go-demo* service.

Next, we'll create the [swarm-listener](https://github.com/vfarcic/docker-flow-swarm-listener) service. It is companion to the `Docker Flow: Proxy`. Its purpose is to monitor Swarm services and send requests to the proxy whenever a service is created or destroyed.

Let's create the `swarm-listener` service.

> ## A note to Windows users
>
> For mounts to work, you will have to enter one of the machines before executing the `docker service create` command to work. To enter the Docker Machine, please execute the command that follows.
>
> `docker-machine ssh node-1`
>
> Please exit the machine once you finish executing the command that follows.
>
> `exit`

```bash
docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener
```

The service is attached to the proxy network, mounts the Docker socket, and declares the environment variables `DF_NOTIF_CREATE_SERVICE_URL` and `DF_NOTIF_REMOVE_SERVICE_URL`. We'll see the purpose of the variables soon. The service is constrained to the `manager` nodes.

The next step is to create the proxy service.

```bash
docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy
```

We opened the ports *80* and *443*. External requests will be routed through them towards destination services. The proxy is attached to the *proxy* network and has the mode set to *swarm*. The proxy must belong to the same network as the listener. They will exchange information whenever a service is created or removed.

Let's deploy the demo service. It consists of two containers; *mongo* is the database and *vfarcic/go-demo* is the actual service that uses it. They will communicate with each other through the *go-demo* network. Since we want to expose only *vfarcic/go-demo* to the "outside" world and keep the database "private", only the *vfarcic/go-demo* container will attach itself to the *proxy* network.

```bash
docker service create --name go-demo-db \
    --network go-demo \
    mongo
```

Let's run up the second service.

```bash
docker service create --name go-demo \
    -e DB=go-demo-db \
    --network go-demo \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/demo \
    --label com.df.port=8080 \
    vfarcic/go-demo
```

The details of the *go-demo* service are irrelevant for this exercise. What matters is that it was deployed somewhere inside the cluster and that it does not have any port exposed outside of the networks *go-demo* and *proxy*.

Please note the labels. They are a crucial part of the service definition. The `com.df.notify=true` tells the `Swarm Listener` whether to send a notifications whenever a service is created or removed. The rest of the labels match the query arguments we would use if we'd reconfigure the proxy manually. The only difference is that the labels are prefixed with `com.df`. For the list of the query arguments, please see the [Reconfigure](../README.md#reconfigure) section.

Now we should wait until all the services are running. You can see their status by executing the command that follows.

```bash
docker service ls
```

Once all the replicas are set to `1/1`, we can see the effect of the `com.df` labels by sending a request to the `go-demo` service through the proxy.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Thu, 13 Oct 2016 18:26:18 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

We sent a request to the proxy (the only service listening to the port 80) and got back the response from the `go-demo` service. The proxy was configured automatically as soon as the `go-demo` service was created.

The way the process works is as follows.

[Docker Flow: Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener) is running inside one of the Swarm manager nodes and queries Docker API in search for newly created services. Once it finds a new service, it looks for its labels. If the service contains the `com.df.notify` (it can hold any value), the rest of the labels with keys starting with `com.df.` are retrieved. All those labels are used to form request parameters. Those parameters are appended to the address specified as the `DF_NOTIF_CREATE_SERVICE_URL` environment variable defined in the `swarm-listener` service. Finally, a request is sent. In this particular case, the request was made to reconfigure the proxy with the service `go-demo` (the name of the service), using `/demo` as the path, and running on the port `8080`. The `distribute` label is not necessary in this example since we're running only a single instance of the proxy. However, in production we should run at least two proxy instances (for fault tolerance) and the `distribute` argument means that reconfiguration should be applied to all.

Please see the [Reconfigure](../README.md#reconfigure) section for the list of all the arguments that can be used with the proxy.

Since *Docker Flow: Proxy* uses new networking features added to Docker 1.12, it redirects all requests to the Swarm SDN (in this case called `proxy`). As a result, Docker takes care of load balancing, so there is no need to reconfigure it every time a new instance is deployed. We can confirm that by creating a few additional replicas.

```bash
docker service update --replicas 5 go-demo

curl -i $(docker-machine ip node-1)/demo/hello
```

Feel free to repeat this request a few more times. Once done, check the logs of any of the replicas and you'll notice that it received approximately one-fifth of the requests. No matter how many instances are running and with which frequency they change, Swarm networking will make sure that requests are load balanced across all currently running instances.

*Docker Flow: Proxy* reconfiguration is not limited to a single *service path*. Multiple values can be divided by comma (*,*). For example, our service might expose multiple versions of the API. In such a case, an example `servicePath` label attached to the `go-demo`service could be as follows.

```bash
...
  --label com.df.servicePath=/demo/hello,/demo/person \
...
```

Optionally, *serviceDomain* can be used as well. If specified, the proxy will allow access only to requests coming from that domain. The example label that follows would set *serviceDomain* to *my-domain.com*. After the proxy is reconfigured, only requests for that domain will be redirected to the destination service.

```bash
...
  --label com.df.serviceDomain=my-domain.com \
...
```

Multiple domains should be separated with comma (`,`).

```bash
...
  --label com.df.serviceDomain=my-domain.com,my-other-domain.com \
...
```

Domains can be prefixed with a wildcard.

```bash
...
  --label com.df.serviceDomain=*domain.com \
...
```

The above example would match any domain ending with `domain.com` (e.g. `my-domain.com`, `my-other-domain.com`, etc).

### Removing a Service From the Proxy

Since `Swarm Listener` is monitoring docker services, if a service is removed, related entries in the proxy configuration will be removed as well.

```bash
docker service rm go-demo
```

If you check the `Swarm Listener` logs, you'll see an entry similar to the one that follows.

```
Sending service removed notification to http://proxy:8080/v1/docker-flow-proxy/remove?serviceName=go-demo
```

A moment later, a new entry would appear in the proxy logs.

```
Processing request /v1/docker-flow-proxy/remove?serviceName=go-demo
Processing remove request /v1/docker-flow-proxy/remove
Removing go-demo configuration
Removing the go-demo configuration files
Reloading the proxy
```

From this moment on, the service *go-demo* is not available through the proxy.

`Swarm Listener` detected that the service was removed, send a notification to the proxy which, in turn, changed its configuration and reloaded underlying HAProxy.

Now that you've seen how to automatically add and remove services from the proxy, let's take a look at scaling options.

### Scaling the Proxy

Swarm is continuously monitoring containers health. If one of them fails, it will be redeployed to one of the nodes. If a whole node fails, Swarm will recreate all the containers that were running on that node. The ability to monitor containers health and make sure that they are (almost) always running is not enough. There is a brief period between the moment an instance fails until Swarm detects that and instantiates a new one. If we want to get close to zero-downtime systems, we must scale our services to at least two instances running on different nodes. That way, while we're waiting for one instance to recuperate from a failure, the others can take over its load. Even that is not enough. We need to make sure that the state of the failed instance is recuperated.

Let's see how *Docker Flow: Proxy* behaves when scaled.

Before we scale the proxy, we'll recreate the `go-demo` service that we removed a few moments ago.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  --replicas 3 \
  vfarcic/go-demo
```

At the moment we are still running a single instance of the proxy. Before we scale it, let's confirm that the listener sent a request to reconfigure it.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output should be status `200` indicating that the proxy works.

Let's scale the proxy to three instances.

```bash
docker service scale proxy=3
```

The proxy was scaled to three instances.

Normally, creating a new instance means that it starts without a state. As a result, the new instances would not have the `go-demo` service configured. Having different states among instances would produce quite a few undesirable effects. This is where the environment variable `LISTENER_ADDRESS` comes into play.

If you go back to the command we used to create the `proxy` service, you'll notice the argument that follows.

```bash
    -e LISTENER_ADDRESS=swarm-listener \
```

This tells the proxy the address of the [Docker Flow: Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener) service. Whenever a new instance of the proxy is created, it will send a request to the listener to resend notifications for all the services. As a result, each proxy instance will soon have the same state as the other.

If, for example, an instance of the proxy fails, Swarm will reschedule it and, soon afterwards, a new instance will be created. In that case, the process would be the same as when we scaled the proxy and, as the end result, the rescheduled instance will also have the same state as any other.

To test whether all the instances are indeed having the same configuration, we can send a couple of requests to the *go-demo* service.

Please run the command that follows a couple of times.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

Since Docker's networking (`routing mesh`) is performing load balancing, each of those requests is sent to a different proxy instance. Each was forwarded to the `go-demo` service endpoint, Docker networking did load balancing and resent it to one of the `go-demo` instances. As a result, all requests returned status *200 OK* proving that the combination of the proxy and the listener indeed works. All three instances of the proxy were reconfigured.

Let's remove the `go-demo` service before we proceed with an example of rewriting paths.

```bash
docker service rm go-demo
```

## Rewriting Paths

In some cases, you might want to rewrite the path of the incoming request before forwarding it to the destination service. We can do that using request parameters `reqRepSearch` and `reqRepReplace`.

As an example, we'll create the `go-demo` service that will be configured in the proxy to accept requests with the path starting with `/something`. Since the `go-demo` service allows only requests that start with `/demo`, we'll use `reqRepSearch` and `reqRepReplace` to rewrite the path.

The command is as follows.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/something \
  --label com.df.port=8080 \
  --label com.df.reqRepSearch='^([^\ ]*)\ /something/(.*)' \
  --label com.df.reqRepReplace='\1\ /demo/\2' \
  --replicas 3 \
  vfarcic/go-demo
```

Please notice that, this time, the `servicePath` is `/something`. The `reqRepSearch` specifies the regular expression that will be used to search for part of the address and the `reqRepReplace` will replace it. In this case, `/something/` will be replaced with `/demo/`. The proxy uses *PCRE compatible regular expressions*. For more information, please consult [Quick-Start: Regex Cheat Sheet](http://www.rexegg.com/regex-quickstart.html) and [PCRE Regex Cheatsheet](https://www.debuggex.com/cheatsheet/regex/pcre).

Please wait a few moments until the `go-demo` service is running. You can see the status of the service by executing `docker service ps go-demo`.

Once the `go-demo` service is up and running, we can confirm that the proxy was indeed configured correctly.

```bash
curl -i $(docker-machine ip node-1)/something/hello
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Sun, 11 Dec 2016 18:43:21 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

We sent a request to `/something/hello`. The proxy accepted the request, rewrote the path to `/demo/hello`, and forwarded it to the `go-demo` service.


Let's remove the `go-demo` service before we proceed with *authentication*.

```bash
docker service rm go-demo
```

## Basic Authentication

*Docker Flow: Proxy* can provide basic authentication that can be applied on two levels. We can configure the proxy to protect all or only a selected service.

### Global Authentication

We'll start by recreating the `go-demo` service we removed earlier.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network go-demo \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  --replicas 3 \
  vfarcic/go-demo
```

To configure the proxy to protect all the services, we need to specify the environment variable `USERS`.

As an example, we'll update the `proxy` service by adding the environment variable `USERS`.

```bash
docker service update --env-add "USERS=my-user:my-pass" proxy
```

Please wait a few moments until all the instances of the `proxy` are updated. You can monitor the status with the `docker service ps proxy` command.

Let's see what will happen if we send another request to the `go-demo` service.

```bash
curl -i $(docker-machine ip node-1)/demo/hello
```

The output is as follows.

```
HTTP/1.0 401 Unauthorized
Cache-Control: no-cache
Connection: close
Content-Type: text/html
WWW-Authenticate: Basic realm="defaultRealm"

<html><body><h1>401 Unauthorized</h1>
You need a valid user and password to access this content.
</body></html>
```

The `proxy` responded with the status `401 Unauthorized`. Our services are not accessible without the username and password.

Let's send one more request but, this time, with the username and password.

```bash
curl -i -u my-user:my-pass \
    $(docker-machine ip node-1)/demo/hello
```

The response is as follows.

```
HTTP/1.1 200 OK
Date: Sun, 04 Dec 2016 23:37:18 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

Since the request contained the correct username and password, proxy let it through and forwarded it to the `go-demo` service.

Multiple usernames and passwords can be separated with a comma.

```bash
docker service update \
    --env-add "USERS=my-user-1:my-pass-1,my-user-2:my-pass-2" \
    proxy
```

Once the update is finished, we will be able to access the services using the user `my-user-1` or `my-user-2`.

```bash
curl -i -u my-user-2:my-pass-2 \
    $(docker-machine ip node-1)/demo/hello
```

As expected, the proxy responded with the status `200` allowing us to access the service.

Let us remove the global authentication before we proceed further.

```bash
docker service update \
    --env-rm "USERS" \
    proxy
```

### Service Authentication

In many cases, we do not want to protect all services but only a selected few. A service can be protected by adding `users` parameter to the `reconfigure` request. Since we are using the [Docker Flow: Swarm Listener](https://github.com/vfarcic/docker-flow-swarm-listener) service to reconfigure the proxy, we'll add the parameter as one more label.

Let's start by removing the `go-demo` service.

```bash
docker service rm go-demo
```

Please wait a few moments until the service is removed and the `swarm-listener` updates the proxy.

Now we can create the `go-demo` service with the label `com.df.users`.

```bash
docker service create --name go-demo \
    -e DB=go-demo-db \
    --network go-demo \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/demo \
    --label com.df.port=8080 \
    --label com.df.users=admin:password \
    vfarcic/go-demo
```

We added the `com.df.users` label with the value `admin:password`. Just as with the global authentication, multiple username/password combinations can be separated with comma (`,`).

After a few moments, the `go-demo` service will be created, and `swarm-lister` will update the proxy.

From now on, the `go-demo` service is accessible only if the username and the password are provided.

```bash
curl -i $(docker-machine ip node-1)/demo/hello

curl -i -u admin:password \
    $(docker-machine ip node-1)/demo/hello
```

The first request should return the status code `401 Unauthorized` while the second went through to the `go-demo` service.

Please note that both *global* and *service* authentication can be combined. In that case, all services would be protected with the users specified through the `proxy` environment variable `USERS` and individual services could overwrite that through the `reconfigure` parameter `users`.

Before we move into the next subject, please remove the service and create it again without authentication.

```bash
docker service rm go-demo
```

Please wait a few moments until the service is removed and the `swarm-listener` updates the proxy.

```bash
docker service create --name go-demo \
    -e DB=go-demo-db \
    --network go-demo \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/demo \
    --label com.df.port=8080 \
    vfarcic/go-demo
```

## Configuring Service SSLs And Proxying HTTPS Requests

Even though we published the proxy port `443` and it is configured to forward traffic to our services, `SSL` communication still does not work. We can confirm that by sending an HTTPS request to our demo service.

```bash
eval $(docker-machine env node-1)

curl -i https://$(docker-machine ip node-1)/demo/hello
```

The output is as follows.

```
curl: (35) Unknown SSL protocol error in connection to 192.168.99.108:-9847
```

The error means that there is no certificate that should be used with HTTPS traffic.

There are two ways we can add certificates to the proxy. One is to create your own Docker image based on `docker-flow-proxy`. `Dockerfile` could be as follows.

```
FROM vfarcic/docker-flow-proxy
COPY my-cert.pem /certs/my-cert.pem
COPY haproxy.tmpl /cfg/tmpl/haproxy.tmpl
```

When the image is built, it will be based on `vfarcic/docker-flow-proxy` and include `my-cert.pem` and `haproxy.tmpl` files. The `my-cert.pem` would be your certificate and the `haproxy.tmpl` the modified version of the original proxy template located in [https://github.com/vfarcic/docker-flow-proxy/blob/master/haproxy.tmpl](https://github.com/vfarcic/docker-flow-proxy/blob/master/haproxy.tmpl). You'd need to replace `{{.CertsString}}` with ` ssl crt /certs/my-cert.pem` (please note that the string should start with space). The complete line would be as follows.

```
    bind *:443 ssl crt /certs/my-cert.pem
```

If your certificate is static (almost never changes) and you are willing to create your own `docker-flow-proxy` image, this might be a good option. As an alternative, certificates can be added to the proxy dynamically through an HTTP request.

For production, you should create your certificate through one of the trusted services. For demo purposes, we'll create a self-signed certificate with `openssl`. Before proceeding, please make sure that `openssl` is installed on your host OS.

We'll store a certificate in `tmp` directory, so let us create it.

```bash
mkdir -p tmp
```

The certificate will be tied to the `*.xip.io` domain, which is handy for demonstration purposes. It'll let us use the same certificate even if our server IP addresses might change while testing locally. With [xip.io](http://xip.io/) don't need to re-create the self-signed certificate when, for example, our Docker machine changes IP.

> I use the [xip.io](http://xip.io/) service as it allows me to use a hostname rather than directly accessing the servers via an IP address. It saves me from editing me computers' host file.

To create a certificate, first we need a key.

```bash
openssl genrsa -out tmp/xip.io.key 1024
```

With the newly created key, we can proceed and create a certificate signing request (CSR). The command is as follows.

```bash
openssl req -new \
    -key tmp/xip.io.key \
    -out tmp/xip.io.csr
```

You will be asked quite a few question. It is important that when "Common Name (e.g. server FQDN or YOUR name)" comes, you answer it with `*.xip.io`. Feel free to answer the rest of the question as you like.

Finally, with the key and the CSR, we can create the certificate. The command is as follows.

```bash
openssl x509 -req -days 365 \
    -in tmp/xip.io.csr \
    -signkey tmp/xip.io.key \
    -out tmp/xip.io.crt
```

As a result, we have the `crt`, `csr`, and `key` files in the `tmp` directory.

Next, after the certificates are created, we need to create a `pem` file. A pem file is essentially just the certificate, the key and optionally certificate authorities concatenated into one file. In our example, we'll simply concatenate the certificate and key files together (in that order) to create a `xip.io.pem` file.

```bash
cat tmp/xip.io.crt tmp/xip.io.key \
    | tee tmp/xip.io.pem
```

To demonstrate how `xip.io` works, we can, for example, send the request that follows.

```bash
curl -i http://$(docker-machine ip node-1).xip.io/demo/hello
```

Our laptop thinks that we are dealing with the domain `xip.io`. When your computer looks up the xip.io domain, the xip.io DNS server extracts the IP address from the domain and sends it back in the response. The [xip.io](http://xip.io/) service is useful for testing purposes.

Now we have the PEM file and a domain we'll use to test it. The only thing missing is to add it to the proxy.

We'll send the proxy service the PEM file we just created. Before we do that, we should publish the port `8080`. It is proxy's internal port we can use to send it commands. If you already have the port `8080` published, there is no need to run the command that follows.

```bash
docker service update \
    --publish-add 8080:8080 proxy
```

Please wait a few moments until the proxy is updated. You can check the status by executing the `docker service ps proxy` command.

Now we can tell the proxy to use the certificate we created. The command is as follows.

```bash
curl -i -XPUT \
    --data-binary @tmp/xip.io.pem \
    "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/cert?certName=xip.io.pem&distribute=true"
```

The `PUT` request to the proxy passed the certificate in its body (`--data-binary @tmp/xip.io.pem`). The request URL has the certification name (`certName`) as one of the parameters. The second parameter (`distribute`) sends the proxy the signal to distribute the certificate to all the replicas. The result of the request is that the certificate has been added to all the replicas of the proxy. We can confirm that by inspecting the proxy config.

```bash
curl $(docker-machine ip node-1).xip.io:8080/v1/docker-flow-proxy/config
```

The relevant part of the output is as follows.

```
frontend services
    bind *:80
    bind *:443 ssl crt /certs/xip.io.pem
    mode http


    acl url_go-demo path_beg /demo
    use_backend go-demo-be if url_go-demo

backend go-demo-be
    mode http
    server go-demo go-demo:8080
```

As you can see, the certificate `xip.io.pem` was added to the `*:443` binding and the proxy is ready to serve HTTPS requests.

Let's confirm that HTTPS works.

```bash
curl -i  \
    https://$(docker-machine ip node-1).xip.io/demo/hello
```

A part of the output is as follows.

```
curl: (60) SSL certificate problem: Invalid certificate chain
```

This time, the error message is different. SSL now works but, since it is self-signed, `curl` cannot verify it. Instead, open it in your favourite browser. The address that should be opened can be obtained with the command that follows.

```bash
echo https://$(docker-machine ip node-1).xip.io/demo/hello
```

On Chrome you'll see "Your connection is not private" message. Click the *ADVANCED* link followed with "Proceed to \[IP\].xip.io (unsafe)" link. You'll see the "hello, world!" message displayed through HTTPS protocol.

Please note that you are not limited to a single certificate. You can send multiple `PUT` requests with different certificates and they will all be added to the proxy.

Now you can secure your proxy communication with SSL certificates. Unless you already have a certificate, purchase it or get it for free from [Let's Encrypt](https://letsencrypt.org/). The only thing left is for you to send a request to the proxy to include the certificate and try it out with your domain.

Before you start using `Docker Flow: Proxy`, you might want to get a better understanding of the flow of a request.

## The Flow Explained

We'll go over the flow of a request to one of the services in the Swarm cluster.

A user or a service sends a request to our DNS (e.g. *acme.com*). The request is usually HTTP on the port `80` or HTTPS on the port `443`.

DNS resolves the domain to one of the servers inside the cluster. We do not need to register all the nodes. A few is enough (more than one in the case of a failure).

The Docker's routing mesh inspects which containers are running on a given port and re-sends the request to one of the instances of the proxy. It uses round robin load balancing so that all instances share the load (more or less) equally.

The proxy inspects the request path (e.g. `/demo/hello`) and sends it the end-point with the same name as the destination service (e.g. `go-demo`). Please note that for this to work, both the proxy and the destination service need to belong to the same network (e.g. `proxy`). The proxy changes the port to the one of the destination service (e.g. `8080`).

The proxy network performs load balancing among all the instances of the destination service, and re-sends the request to one of them.

The whole process sounds complicated (it actually is from the engineering point of view). But, as a user, all this is transparent.

One of the important things to note is that, with a system like this, everything can be fully dynamic. Before the new Swarm introduced in Docker 1.12, we would need to run our proxy instances on predefined nodes and make sure that they are registered as DNS records. With the new routing mesh, it does not matter whether the proxy runs on a node registered in DNS. It's enough to hit any of the servers, and the routing mesh will make sure that it reaches one of the proxy instances.

A similar logic is used for the destination services. The proxy does not need to do load balancing. Docker networking does that for us. The only thing it needs is the name of the service and that both belong to the same network. As a result, there is no need to reconfigure the proxy every time a new release is made or when a service is scaled.

## Usage

Please explore [Usage](../README.md#usage) for more information.
