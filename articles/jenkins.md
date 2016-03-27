A few days ago I released [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project aimed at providing a simple way to reconfigure proxy every time a new service is deployed or when a service is scaled. For more information, please visit the GitHub repository [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) or read the [Docker Flow: Proxy – On-Demand HAProxy Service Discovery and Reconfiguration](http://technologyconversations.com/2016/03/21/docker-flow-proxy-on-demand-haproxy-service-discovery-and-reconfiguration/) article. Considering that you are reading this post on the [CloudBees blog](https://www.cloudbees.com/blog), I'll assume that you are a Jankins fan and that you are using it to deploy your Docker containers. In that spirit, what follows is an example that uses [Jenkins Pipeline](https://wiki.jenkins-ci.org/display/JENKINS/Pipeline+Plugin) to deploy a service to a Docker Swarm cluster and automatically reconfigure the proxy so that the users of the service can access it no matter its location(s) or the number of scaled instances.

Setting Up the Environments
===========================

The only prerequisites for the example we are about to explore are [Vagrant](https://www.vagrantup.com/) and [Git client](https://git-scm.com/).

Let's start by checking out the project code.


```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

To demonstrate the benefits of *Docker Flow: Proxy*, we'll set up a Swarm cluster and deploy a service. We'll create four virtual machines. One (*jenkins*) will be running *Jenkins*, *Docker Flow: Proxy*, and *Consul*. The other three machines will form the Swarm cluster with one master and two nodes.

Since I want to concentrate on how *Docker Flow: Proxy* works, I'll skip detailed instructions how will the environments be set up and only mention that we'll use [Ansible](https://www.ansible.com/) to provision the VMs.

Before we spin up the machines, you might want to install the *vagrant-cachier* plugin. It will cache dependencies so that subsequent package installations are done much faster.

```
vagrant plugin install vagrant-cachier
```

Let us create the VMs.

```bash
vagrant up swarm-master swarm-node-1 swarm-node-2 jenkins
```

Creating and provisioning four servers requires a bit of time so grab some coffee and come back in a few minutes. We'll continue when `vagrant` command is finished.

Now we have the four servers up and running. The first one (*jenkins*) is running [Consul](https://www.consul.io/), *docker-flow-proxy*, and *Jenkins* containers. Consul will contain all the information we might need for proxy configuration. At the same time, it is a service discovery tool of choice for setting up a Swarm cluster. In production, you should probably run it on all servers in the cluster but, for this demo, one instance should do.

The other three VMs constitute the cluster. Besides Swarm itself, each of the machines in the cluster is running *Registrator* that monitors Docker events and puts data into Consul whenever a new container is run. It works the other way as well. If a container is stopped or removed, Registrator will eliminate the data from Consul. In other words, thanks to Registrator, Consul will always have up-to-date information on all containers running on the cluster.

Let's enter the *jenkins* VM and take a quick look at the cluster status.

```bash
vagrant ssh jenkins

export DOCKER_HOST=tcp://10.100.192.200:2375

docker info
```

The output of the `docker info` is as follows.

```
Containers: 4
 Running: 4
 Paused: 0
 Stopped: 0
Images: 4
Server Version: swarm/1.1.3
Role: primary
Strategy: spread
Filters: health, port, dependency, affinity, constraint
Nodes: 2
 swarm-node-1: 10.100.192.201:2375
  └ Status: Healthy
  └ Containers: 2
  └ Reserved CPUs: 0 / 1
  └ Reserved Memory: 0 B / 1.536 GiB
  └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-79-generic, operatingsystem=Ubuntu 14.04.4 LTS, storagedriver=devicemapper
  └ Error: (none)
  └ UpdatedAt: 2016-03-21T18:45:44Z
 swarm-node-2: 10.100.192.202:2375
  └ Status: Healthy
  └ Containers: 2
  └ Reserved CPUs: 0 / 1
  └ Reserved Memory: 0 B / 1.536 GiB
  └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-79-generic, operatingsystem=Ubuntu 14.04.4 LTS, storagedriver=devicemapper
  └ Error: (none)
  └ UpdatedAt: 2016-03-21T18:45:06Z
Plugins:
 Volume:
 Network:
Kernel Version: 3.13.0-79-generic
Operating System: linux
Architecture: amd64
CPUs: 2
Total Memory: 3.072 GiB
Name: 29d99e7c2b64
```

As you can see, the Swarm cluster consists of two nodes (*swarm-node-1* and *swarm-node-2*), each has one CPU and 1.5 GB of RAM, and the status is *Healthy*.

To summarize, we set up four servers. The *jenkins* node hosts *Docker Flow: Proxy*, *Jenkins*, and *Consul*. *Docker Flow: Proxy* will be our single entry into the system, and *Consul* will act as service registry. All requests to our services will go to a single port *80* in the *proxy* node, and HAProxy will make sure that they are redirected to the final destination.

The other three nodes constitute our Docker Swarm cluster. The *swarm-master* is in charge of orchestration and will deploy services to one of its nodes (at the moment only *swarm-node-1* and *swarm-node-2*). Each of those nodes is running *Registrator* that monitors Docker events and updates *Consul* information about deployed (or stopped) containers.

Now that everything is set up let's deploy a few services.

Running a Single Instance of a Service
======================================

Since I'm a huge fun of everything-described-as-a-code approach, I choose to write the jobs we'll explore through the [Jenkins Pipeline](https://wiki.jenkins-ci.org/display/JENKINS/Pipeline+Plugin) plugin.

We'll start by exploring the [deploy-to-swarm-without-proxy](http://10.100.199.200:8080/job/deploy-to-swarm-without-proxy) job. Please open the [deploy-to-swarm-without-proxy/configure](http://10.100.199.200:8080/job/deploy-to-swarm-without-proxy/configure) screen. The Pipeline script is as follows.

```groovy
node("docker") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    withEnv(["DOCKER_HOST=tcp://10.100.192.200:2375"]) {
        sh "docker-compose -p books-ms -f docker-compose-demo.yml up -d"
    }
}
```

As you can see, this is a very simple script. It defines that the pipeline should run inside the *docker* node and clones the code from the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) repository. It contains a block that should be executed with `DOCKER_HOST` variable set to the address of the *swarm-master*. Within that block, we're running the *Docker Compose* command that will bring up all the containers defined in the *docker-compose-demo.yml* file. We could have used the [CloudBees Docker Pipeline Plugin](https://wiki.jenkins-ci.org/display/JENKINS/CloudBees+Docker+Pipeline+Plugin) plugin instead the Compose. Discussion about advantages and disadvantages of using one way over the other is beyond the scope of this article.

Let's build the job and see the result. Please click the *Build Now* button, open the [last build console screen](http://10.100.199.200:8080/job/deploy-to-swarm-without-proxy/lastBuild/console), and wait until the build is finished (please note that the first time it might take a while).

We just run a service that exposes HTTP API. The details of the service are not important for this article. What matters is that it is running on a random port exposed by Docker Engine. Since we're using Swarm, it might be running on any of its nodes. In other words, both IP and port of the service are determined by Swarm instead being controlled by us. That is a good thing since Swarm takes away tedious task of managing services in a (potentially huge) cluster. However, not knowing IP and port in advance poses a few questions, most important one being how to access the service if we don't know where it is.

This is the moment when *Registrator* comes into play. It detected that a new container is running and stored its data into Consul. We can confirm that by running the following request.

```bash
curl consul:8500/v1/catalog/service/books-ms \
    | jq '.'
```

The output of the `curl` command is as follows.

```
[
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.202",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-2:booksms_app_1:8080",
    "Address": "10.100.199.200",
    "Node": "jenkins"
  }
]
```

We can see that, in this case, the service is running in *10.100.192.202* (*swarm-node-2*) on the port *32768*. The name of the service (*books-ms*) is the same as the name of the container we deployed.

The fact that our service was deployed somewhere inside the cluster does not mean that it is accessible to our users. As it is now, the service is useless. What we're missing is integration with the proxy service. That is the part when *Docker Flow: Proxy* can help.

Please open the [deploy-to-swarm-with-proxy configuration screen](http://10.100.199.200:8080/job/deploy-to-swarm-with-proxy/configure). You'll notice that the script is slightly different.

```groovy
node("docker") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    withEnv(["DOCKER_HOST=tcp://10.100.192.200:2375"]) {
        sh "docker-compose -p books-ms -f docker-compose-demo.yml up -d"
    }
    sh "curl \"proxy:8081/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books\""
}
```

The difference is in the last line that sends an HTTP request to the *Docker Flow: Proxy* API. We'll discuss the format of the request shortly. Right now, we'll run the build.

Please click the *Build Now* button, open the [console screen of the last build](http://10.100.199.200:8080/job/deploy-to-swarm-with-proxy/lastBuild/console), and wait until it's finished.

That's it. All we had to do is send an HTTP request to `reconfigure` the proxy. The `serviceName` query contains the name of the service we want to integrate with the proxy. It needs to match the *ServiceName* value stored in Consul. The `servicePath` is the unique URL that identifies the service. HAProxy will redirect all requests with URL that begin with that value.

Let's see whether the service is indeed accessible through the proxy.

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

The response is *200 OK*, meaning that our service is indeed accessible through the proxy. All we had to do is tell *docker-flow-proxy* the name of the service and its base URL.

Scaling The Service
===================

*Docker Flow: Proxy* is not limited to a single instance. It will reconfigure proxy to perform load balancing among all currently deployed instances of a service.

Let's take a look at another sample job. Please open the [scale-to-swarm-with-proxy configuration screen](http://10.100.199.200:8080/job/scale-to-swarm-with-proxy/configure).

You'll notice that it has a single parameter called *SCALE* and the Pipeline script that is as follows.

```groovy
node("docker") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    withEnv(["DOCKER_HOST=tcp://10.100.192.200:2375"]) {
        sh "docker-compose -p books-ms -f docker-compose-demo.yml scale app=${SCALE}"
    }
    sh "curl \"proxy:8081/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&servicePath=/api/v1/books\""
}
```

This time, instead running `docker-compose up`, we're executing the `docker-compose scale` command. The number of instances we'd like to scale to is specified through the *SCALE* variable.

Please click the *Build With Parameters* link, type *3* in the *SCALE* field, and click the *Build* button. You can monitor the progress by opening the [last build console screen](http://10.100.199.200:8080/job/scale-to-swarm-with-proxy/lastBuild/console).

This time, three instances of the service are running. *Docker Swarm* made sure that they are distributed throughout the cluster and *Docker Flow: Proxy* reconfigured the HAProxy.

We can confirm that the service is, indeed, scaled by running `docker ps` command.

```bash
docker ps -a --filter name=books --format "table {{.Names}}"
```

The output of the `docker ps` command is as follows.

```
NAMES
swarm-node-2/booksms_app_2
swarm-node-1/booksms_app_3
swarm-node-1/books-ms-db
swarm-node-2/booksms_app_1
```

As you can see, four containers (one belonging to the database and three instances of the service) are distributed across the Swarm cluster (two containers on each server). We can observe the same by querying Consul.

```bash
curl consul:8500/v1/catalog/service/books-ms \
    | jq '.'
```

This time, Consul returned different results.

```
[
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.202",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-2:booksms_app_1:8080",
    "Address": "10.100.199.200",
    "Node": "jenkins"
  },
  {
    "ServicePort": 32769,
    "ServiceAddress": "10.100.192.202",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-2:booksms_app_2:8080",
    "Address": "10.100.199.200",
    "Node": "jenkins"
  },
  {
    "ServicePort": 32768,
    "ServiceAddress": "10.100.192.201",
    "ServiceTags": null,
    "ServiceName": "books-ms",
    "ServiceID": "swarm-node-1:booksms_app_3:8080",
    "Address": "10.100.199.200",
    "Node": "jenkins"
  }
]
```

Finally, we can double check that the proxy is still working.

```
curl -I jenkins/api/v1/books
```

From this moment on, *HAProxy* is configured to perform load balancing across all three instances. We can continue scaling (and de-scaling) the service and, as long as we send the `reconfigure` request, the proxy will load-balance requests across all instances. They can be distributed among any number of servers, or even across different datacenters (as long as they are accessible from the proxy server).

Please give *Docker Flow: Proxy* a try. Deploy multiple services, scale them, destroy them, and so on. More information can be found in the project [README](https://github.com/vfarcic/docker-flow-proxy).

Before moving onto a next task, please do not forget to stop (or destroy) VMs we created and free your resources.

```bash
exit

vagrant destroy -f
```
