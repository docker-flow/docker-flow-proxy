# Debugging Docker Flow Proxy

*Docker Flow Proxy* is designed for high loads. One of the conscious design decision was to limit logging to a minimum. By default, you will see logs only for the events sent to the proxy, not from user's requests destined to your services. If logging of all requests is enabled, it could require more resources than request forwarding (proxy's primary function).

While the decision to provide minimal logging is a good one when things are working correctly, you might find yourself in a situation when the proxy is not behaving as expected. In such a case, additional logging for a limited time can come in handy.

The examples that follow will show you how to enable *Docker Flow Proxy* debugging mode.

## Creating a Swarm Cluster

!!! tip
    Feel free to skip this section if you already have a working Swarm cluster.

!!! info
	If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox* or *Git*).

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

Now that we have a Swarm cluster, we can deploy *Docker Flow Proxy* stack together with a demo service.

## Deploying Docker Flow Proxy And a Demo Service

!!! tip
    You might already have the `proxy` and `go-demo` services deployed from one of the other tutorials. If that's the case, feel free to skip this section.

```bash
docker network create -d overlay proxy

curl -o proxy.yml \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c proxy.yml proxy

curl -o go-demo.yml \
    https://raw.githubusercontent.com/vfarcic/go-demo/master/docker-compose-stack.yml

docker stack deploy -c go-demo.yml go-demo
```

Please wait until the `go-demo` service is running. You can see the status by executing the `docker stack ps go-demo` command. Please note that, since `go-demo_main` depends on the `go-demo_db` service, you might see a few failures until the latter is pulled and running. The end state should be three replicas of `go-demo_main` with the current state set to running.

Let's explore the proxy logs that come as default.

## Logging Without The Debug Mode

We'll send two requests to the `proxy`.

```bash
curl -i "http://$(docker-machine ip node-1)/demo/hello"

curl -i "http://$(docker-machine ip node-1)/this/endpoint/does/not/exist"
```

Since the endpoint of the second request does not exist, the response status is `503`. What we don't know is whether we got that response because there's something wrong with the service, because it is not configured in the proxy, or, maybe, because of some completely unrelated reason.

Let's take a look at `proxy` logs.

```bash
docker service logs proxy_proxy
```

We can see log entries from the requests sent by `swarm-listener`, but there is no trace of the two requests we made. We need to enable debugging.

## Logging With The Debug Mode

By default, debugging is disabled for a reason. It slows down the proxy. While that might not be noticeable in this demo, when working with thousands of requests per second, debugging can prove to be a bottleneck.

!!! danger
	Do not enable debugging in production.

We'll start by updating the `proxy` service.

```bash
docker service update --env-add DEBUG=true proxy_proxy
```

We added the environment variable `DEBUG` set to `true`. From now on proxy will send to `stdout` information about each request.

Let's take a look at the logs.

!!! info
	The command that follows uses `docker service logs` command that is (at the time of this writing) still in experimental stage.

If you used an existing cluster, `docker service logs` might not be available, and you might need to look at the logs of a container instead.

```bash
docker service logs -f proxy_proxy
```

Since we used the `-f` flag, we are following the logs and won't be able to use the same terminal for anything else. Please open a new terminal window.

```bash
eval $(docker-machine env node-1)

curl -i "http://$(docker-machine ip node-1)/demo/hello"

curl -i "http://$(docker-machine ip node-1)/this/endpoint/does/not/exist"
```

We repeated the same two requests we used before.

Please go back to the other terminal and observe the logs.

The relevant part of the output is as follows.

```
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 150 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/hello HTTP/1.1"
HAPRoxy: services services/<NOSRV> -1/-1/-1/-1/0 503 1271 - - SC-- 0/0/0/0/0 0/0 {-,"",""} "GET /this/endpoint/does/not/exist HTTP/1.1"
```

As you can see, both requests were recorded.

Let's do a bit more sophisticated demo.

We'll send twenty requests to the demo service endpoint that randomly responds with errors.

```bash
for i in {1..20}
do
    curl "http://$(docker-machine ip node-1)/demo/random-error"
done
```

The output with only relevant parts is as follows.

```
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 500 196 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 500 196 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/0/0 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/1/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 500 196 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
HAPRoxy: services go-demo_main-be8080/go-demo_main 0/0/0/1/1 200 159 - - ---- 1/1/0/0/0 0/0 {-,"",""} "GET /demo/random-error HTTP/1.1"
```

Approximately, one out of ten responses returned status code `500`.

Logs contain quite a lot of other useful information. I suggest you consult [Debug Format](#debug-format) for a complete description of the output.

We are finished with the short introduction to *Docker Flow Proxy* debugging feature. We should destroy the demo cluster and free our resources for something else.

```bash
docker-machine rm -f $(docker-machine ls -q)
```

If you used your own cluster, hopefully, it was a testing environment. If it wasn't, please remove the `DEBUG` label by executing the command that follows.

```bash
docker service update --env-rm DEBUG proxy_proxy
```