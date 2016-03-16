The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide an easy way to reconfigure proxy every time a new service is deployed or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and join them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) for service discovery. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

Instead of debating theory, let's see it in action. We'll start by setting up an example environment.

Setting Up the Environments
===========================

The only prerequisites for this article are [Vagrant](https://www.vagrantup.com/) and [Git client](https://git-scm.com/).

Let's start by checking out the project code.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

To demonstrate the benefits of *Docker Flow: Proxy*, we'll setup a Swarm cluster and deploy a few services. We'll create four virtual machines. One (*proxy*) will be running *Docker Flow: Proxy*. The other three machines will form a Swarm cluster with one master and two nodes.

Since I want to concentrate on how *Docker Flow: Proxy* works, I'll skip detailed instructions how the environments will be set up and only mention that we'll use [Ansible](https://www.ansible.com/) to provision the VMs. Besides Docker Engine, the only requirement for *Docker Flow: Proxy* is that service information (IPs and ports) are store in [Consul](https://www.consul.io/).

```
vagrant up proxy swarm-master swarm-node-1 swarm-node-2
```

Now we have the four servers up and running. The first one (*proxy*) is already provisioned so let us setup the Swarm cluster on the other three servers.

Besides Swarm itself, we'll run [Registrator](https://github.com/gliderlabs/registrator) and [Consul](https://www.consul.io/) on all nodes of the cluster. Registrator will monitor Docker events and store service information we need in Consul.

```bash
vagrant ssh proxy

ansible-playbook \
    /vagrant/ansible/swarm.yml \
    -i /vagrant/ansible/hosts/prod
```

Let's take a quick look at the cluster status.

```bash
export DOCKER_HOST=tcp://10.100.192.200:2375

docker info
```

The output of the `docker info` command should show that two nodes are managed by Swarm master.

To summarize, we set up four servers. The *proxy* node hosts *Docker Compose*, *Consul*, and *Docker Flow: Proxy*. We'll use Docker Compose as a convenient way to run containers, Consul will have information about currently running services, and *Docker Flow: Proxy* will be our single entry into the system. All requests to our services will go to a single address, and HAProxy will make sure that they are redirected to the final destination.

The other three nodes constitute our Docker Swarm cluster. The *swarm-master* is in charge of orchestration and will deploy services to one of its nodes (at the moment only *swarm-node-1* and *swarm-node-2*).

Now that everything is set up, let's run a few services.

Running a Single Instance of a Service
======================================

We'll run services defined in the [docker-compose-demo.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-demo.yml).

```bash
cd /vagrant

export DOCKER_HOST=tcp://swarm-master:2375

docker-compose \
    -f docker-compose-demo.yml \
    up -d
```

We just run a service that exposes HTTP API. The details of the service are not important for this article. What does matter is that it is running on a random port exposed by Docker Engine. Since we're using Swarm, it might be running on any of its nodes. In other words, both IP and port of the service is unknown. This is a good thing since Swarm takes away tedious task of managing services in a (potentially huge) cluster. However, not knowing IP and port in advance poses a few questions, most important one being how to access the service if we don't know where it is?

This is the moment when *Registrator* comes into play. It detected that a new container is running and stored its data into Consul. We can confirm that by running the following request.

```bash
curl swarm-master:8500/v1/catalog/service/books-ms \
    | jq '.'
```

The output of the command is as follows.

```json
[
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.201",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-1:vagrant_app_1:8080",
    "Address": "172.17.0.2",
    "Node": "6829011907a8"
  }
]
```

We can see that, in this case, the service is running in *10.100.192.201* on the port *32768*. The name of the service (*books-ms*) is the same as the name of the container we deployed. All we have to do now is reload the proxy.

```bash
export DOCKER_HOST=tcp://proxy:2375

docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

That's it. That single command reconfigured the proxy. All we had to do is run the `reconfigure` command together with a few arguments. The `--service-name` contains the name of the service we want to integrate with the proxy. The `--service-path` is the unique URL that identifies the service.

Let's see whether it worked.

```bash
curl -I proxy/api/v1/books
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

Scaling The Service
===================

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances.

As an example, let's scale the service to three instances.

```bash
export DOCKER_HOST=tcp://swarm-master:2375

docker-compose \
    -f docker-compose-demo.yml \
    scale app=3
```

Let's see the result.

```bash
docker-compose -f docker-compose-demo.yml ps
```

The result of the `docker-compose ps` command is as follows.

```
    Name               Command          State               Ports
------------------------------------------------------------------------------
books-ms-db     /entrypoint.sh mongod   Up      27017/tcp
vagrant_app_1   /run.sh                 Up      10.100.192.201:32768->8080/tcp
vagrant_app_2   /run.sh                 Up      10.100.192.202:32768->8080/tcp
vagrant_app_3   /run.sh                 Up      10.100.192.201:32769->8080/tcp
```

As you can see, the service is scaled to three instances (not counting the database). Two of them are running on *swarm-node-1* (*10.100.192.201*) while the third is in *swarm-node-2* (*10.100.192.202*). Even though three instances are running, the proxy continues redirecting all requests to the first instances. We can change that by re-running the `reconfigure` command.

```bash
export DOCKER_HOST=tcp://proxy:2375

docker exec docker-flow-proxy \
    docker-flow-proxy reconfigure \
    --service-name books-ms \
    --service-path /api/v1/books
```

From this moment on, HAProxy is reconfigured to perform load balancing across all three instances. We can continue scaling (and de-scaling) the service and, as long as the `reconfigure` command is run, the proxy will load-balance all the requests. Those instances can be distributed among any number of servers, or even across different datacenters (as long as they are accessible from the proxy server).

Please give *Docker Flow: Proxy* a try. Deploy multiple services, scale them, destroy them, and so on. The project [README](https://github.com/vfarcic/docker-flow-proxy) has more information. If you have a problem, suggestion, or an opinion regarding the project, please send me an email (my info is in the [About](http://technologyconversations.com/about/) section) or create a [New Issue](https://github.com/vfarcic/docker-flow-proxy/issues).

Before moving onto a next task, please do not forget to stop (or destroy) VMs we created and free your resources.

```bash
vagrant halt
```
