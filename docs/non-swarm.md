# Configuring Non-Swarm Services

While we might want to aim at having all our services inside the same Swarm cluster, there are cases when that is not feasible. We might have multiple Swarm clusters, we might deploy services as "normal" containers, or we might have legacy applications that are not yet ripe for dockerization. No matter the deployment method, we can, still, leverage *Docker Flow Proxy* features to include those non-Swarm services.

The examples that follow will show you how to reconfigure *Docker Flow Proxy* manually.

!!! info
	If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox* or *Git*).

## Creating a Swarm Cluster

!!! tip
    Feel free to skip this section if you already have a working Swarm cluster.

We'll use the [swarm-cluster.sh](https://github.com/vfarcic/docker-flow-proxy/blob/master/scripts/swarm-cluster.sh) script from the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) repository. It'll create a Swarm cluster based on three Docker Machine nodes.

!!! info
	For the [swarm-cluster.sh](https://github.com/vfarcic/docker-flow-proxy/blob/master/scripts/swarm-cluster.sh) script to work, you are required to have Docker Machine installed on your system.

```bash
curl -o swarm-cluster.sh \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/scripts/swarm-cluster.sh

chmod +x swarm-cluster.sh

./swarm-cluster.sh

eval $(docker-machine env node-1)
```

Now that we have a Swarm cluster, we can deploy *Docker Flow Proxy* stack.

## Deploying Docker Flow Proxy

!!! tip
    You might already have the `proxy` stack deployed from one of the other tutorials. If that's the case, feel free to skip this section.

```bash
docker network create -d overlay proxy

curl -o proxy.yml \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c proxy.yml proxy
```

Please wait until the `proxy` stack is running. You can see the status by executing the `docker stack ps proxy` command.

Now we are ready to explore a few ways to configure the proxy to forward requests to services that are not running inside the Swarm cluster we created.

## Manually Configuring Non-Swarm Services

We'll start by creating a new node that will operate outside the Swarm cluster we set up.

```bash
docker-machine create -d virtualbox not-swarm

eval $(docker-machine env not-swarm)
```

The node we created is a simulation of a server that is not part of the Swarm cluster where the proxy is running. In your case, it could be an on-premise physical server, a VMWare virtual machine, an AWS EC2 node, or any other server type.

Now we can deploy a demo service into our non-Swarm node.

```bash
docker network create -d bridge go-demo

docker container run -d --name go-demo-db \
    --network go-demo \
    mongo

docker container run -d --name go-demo \
    --network go-demo \
    --restart on-failure \
    -p 8080:8080 \
    -e DB=go-demo-db \
    vfarcic/go-demo
```

We run two containers (`go-demo-db` and `go-demo`) and connected them through the `bridge` network. If you followed some of the other tutorials, you should already be familiar with those services.

Since the `go-demo` service is not running inside the Swarm cluster and cannot leverage its Overlay network, we had to expose the port `8080`.

!!! info
    This is only a simulation of non-Swarm services. The logic that follows would be the same no matter the deployment method used for your services.

Let's double-check that the `go-demo` service works as expected.

```bash
curl -i "http://$(docker-machine ip not-swarm):8080/demo/hello"
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Tue, 14 Mar 2017 12:50:30 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

The service responded with the status code `200` indicating that we can, indeed, access it.

Let's check whether it is accessible from the proxy (I hope you already know the answer).

```bash
curl -i "http://$(docker-machine ip node-1)/demo/hello"
```

The output is as follows.

```
HTTP/1.0 503 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>503 Service Unavailable</h1>
No server is available to handle this request.
</body></html>
```

Since the `go-demo` service is not running as a service inside the same Swarm cluster as the `proxy` service, `swarm-listener` could not detected it, so the proxy is left oblivious of its existence.

Fortunately, we can send a request to the proxy with the information it needs to reconfigure itself and include the `go-demo` service into its forwarding rules.

If all the services are running inside the same Swarm cluster, our `proxy` stack would work well as it is. *Docker Flow Swarm Listener* would detect new or updated services and send requests to the proxy. Those requests would be sent through the internal Overlay network. However, since we'll reconfigure the proxy manually, we'll need to open the proxy port `8080`.

```bash
eval $(docker-machine env node-1)

docker service update \
    --publish-add 8080:8080 \
    proxy_proxy
```

Now that the port `8080` is opened, we can send the proxy a `reconfigure` request with the information about the `go-demo` service running outside the cluster.

```bash
curl "$(docker-machine ip node-1):8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&servicePath=/demo&port=8080&outboundHostname=$(docker-machine ip not-swarm)&distribute=true"
```

The proxy responded with the status `OK` indicating that it received the `reconfigure` request. The request itself contains the name of the service (`serviceName=go-demo`), the path (`servicePath=/demo`), the port (`port=8080`), the IP of the node (`outboundHostname=$(docker-machine ip not-swarm)`), and the flag that tells it to distribute that information to all the replicas (`distribute=true`).

Please consult the [Reconfigure](usage/#reconfigure) documentation for more information about the parameters we used.

Let's verify that the proxy was indeed reconfigured correctly.

```bash
curl -i "http://$(docker-machine ip node-1)/demo/hello"
```

The output is as follows.

```
HTTP/1.1 200 OK
Date: Tue, 14 Mar 2017 13:19:56 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

The `go-demo` service responded with the status `200` proving that the integration with the proxy was successful.

Even though reconfiguring the proxy manually fulfilled our need to include external services, the method we used has a huge drawback.

## Failover Of Manually Configured Proxy Replicas

In case one of the `proxy` replicas fails, Swarm will schedule a new one. Since the `go-demo` service is not part of the Swarm cluster, the new replica will not be able to request information about the `go-demo` service from the `swarm-listener`. As the result, it will not be configured to forward requests to `go-demo`. A similar scenario would happen if we decide to scale the proxy.

Let's demonstrate the problem with a few commands.

We'll simulate a failure of a replica by scaling down to one instance and then scaling back to two.

```bash
docker service scale proxy_proxy=1
```

We should wait for a second until Swarm becomes aware of the new number of replicas and proceed to upscale the service back to two instances.

```bash
docker service scale proxy_proxy=2
```

What would happen if we send another request to the `go-demo` service through the proxy?

One replica of the proxy still has the `go-demo` configuration while the other (the new one) is oblivious about it. We can demonstrate this situation by sending the same request we used before.

```bash
curl -i "http://$(docker-machine ip node-1)/demo/hello"
```

Since one proxy replica is configured properly and the other one isn't, every second request will have the output that follows.

```
HTTP/1.0 503 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>503 Service Unavailable</h1>
No server is available to handle this request.
</body></html>
```

How can we fix this?

## Configuring Non-Swarm Services Through Environment Variables

We can configure non-Swarm services through environment variables. Unlike manual requests, those variables will be persistent in case of a failure of a replica as well as when we scale the proxy service.

Let's try it out.

```bash
docker service update \
    --env-add DFP_SERVICE_1_SERVICE_NAME=go-demo \
    --env-add DFP_SERVICE_1_SERVICE_PATH=/demo \
    --env-add DFP_SERVICE_1_PORT=8080 \
    --env-add DFP_SERVICE_1_OUTBOUND_HOSTNAME=$(docker-machine ip not-swarm) \
    --publish-rm 8080 \
    proxy_proxy
```

We updated the `proxy` service by adding the `go-demo` information as environment variables. Naming of those variables follows a simple pattern. The are all upper cased, words are separated with an underscore (`_`) and they are prefixed with `DFP_SERVICE` followed with an index. Since we specified only one service, all the variables are prefixed with `DFP_SERVICE_1`. The second service would have variables with the prefix `DFP_SERVICE_2`, and so on.

Additionally, we closed the port `8080`. Since we won't issue any new HTTP `reconfigure` requests, we don't need the port any more.

!!! tip
    We issued a `docker service update` command only to quickly demonstrate how the proxy works with the `DFP_SERVICE` environment variables. You should put them into your stack YAML file instead.

Let's check whether the proxy is configured correctly.

```bash
curl -i "http://$(docker-machine ip node-1)/demo/hello"
```

As expected, the response code is `200` indicating the the proxy forwards the requests to the `go-demo` service. Feel free to re-send the request and confirm that the second replica is configured correctly as well.

Finally, we'll validate that the configuration is preserved in case of a failure.

```bash
docker service scale proxy_proxy=1

docker service scale proxy_proxy=2

curl -i "http://$(docker-machine ip node-1)/demo/hello"
```

We repeated the same simulation of a failure by scaling down to one instance followed with the upscale to two instances. No matter how many times we send the request to the `go-demo` service through the proxy, the result is always the response status `200`.

## What Now?

By itself, manual reconfiguration of the proxy is not fault tolerant. There are quite a few way to fix that.

One of the way we can persist the information about non-Swarm services is to use network volumes to preserve the state of the proxy in case of a failure. While that is a viable option, using environment variables is much easier. We should specify them as part of a YAML file that defines the proxy stack.

Please consult [Reconfigure > Environment Variables](http://proxy.dockerflow.com/usage/#environment-variables) section of the documentation for more information about the environment variables we can use to reconfigure the proxy.

We are finished with the short introduction to *Docker Flow Proxy* configuration with non-Swarm services. We should destroy the demo cluster and free our resources for something else.

```bash
docker-machine rm -f $(docker-machine ls -q)
```
