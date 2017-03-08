# Debugging Docker Flow Proxy

The proxy is designed for high loads and one of the conscious design decision is to limit logging to a minimum. As such, by default, you will see logs only for the events sent to the proxy, not from user's requests destined to your services. If logging of all requests would be enabled, logging often requires more resources than request forwarding (proxy's primary function).

While the decision to provide minimal logging is a good one when things are working correctly, you might find yourself in a situation when the proxy is not behaving as expected. In such a case, additional logging can come in handy for a limited time.

The examples that follow will show you how to enable *Docker Flow Proxy* debugging mode.

## Create a Swarm Cluster

Feel free to skip this section if you already have a working Swarm cluster.

!!! info
	If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox*). Also, make sure that your Git client is configured to check out the code *AS-IS*. Otherwise, Windows might change carriage returns to the Windows format.

We'll use the [swarm-cluster.sh](https://github.com/vfarcic/docker-flow-proxy/blob/master/scripts/swarm-cluster.sh) from the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) repository. It'll create a Swarm cluster based on three Docker Machine nodes.

!!! info
	For the [swarm-cluster.sh](https://github.com/vfarcic/docker-flow-proxy/blob/master/scripts/swarm-cluster.sh) script to work, you are required to have Docker Machine installed on your system.

```bash
curl -o swarm-cluster.sh \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/scripts/swarm-cluster.sh

chmod +x swarm-cluster.sh

./swarm-cluster.sh

eval $(docker-machine env node-1)
```

## Deploy Docker Flow Proxy

```bash
docker network create -d overlay proxy

curl -o proxy.yml \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c proxy.yml proxy

curl -o go-demo.yml \
    https://raw.githubusercontent.com/vfarcic/go-demo/master/docker-compose-stack.yml

docker stack deploy -c go-demo.yml go-demo

docker stack ps go-demo # Wait until it's running
```

## Without Debugging

```bash
curl "http://$(docker-machine ip node-1)/demo/hello"

curl "http://$(docker-machine ip node-1)/this/endpoint/does/not/exist"

docker service logs proxy_proxy
```

## Enable Debugging

!!! danger
	XXX

```bash
docker service update --env-add DEBUG=true proxy_proxy

docker stack ps proxy

docker service logs -f proxy_proxy

# Open a separate terminal

eval $(docker-machine env node-1)

curl "http://$(docker-machine ip node-1)/demo/hello"

curl "http://$(docker-machine ip node-1)/this/endpoint/does/not/exist"

for i in {1..20}
do
    curl "http://$(docker-machine ip node-1)/demo/random-error"
done
```