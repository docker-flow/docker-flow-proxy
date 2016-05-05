Running Tests Inside Docker Containers
======================================

I was involved in many automation projects and come to realize that the key stumbling block towards extensive automation and continuous delivery or deployment (CD) is unclear separation of concerns. This time, by separation of concerns, I'm not referring to design principle for separating a computer program into distinct sections but separation based on different roles people have within organization they work in as well as tasks they perform. This becomes even more evident when we adopt microservices architecture and containers. There is no real reason, any more, not to let people inside the project be in charge of the whole deployment pipeline. When we start doing hand-overs to others (people not working with us on day-to-day basis) we start having delays. The problem is that small teams (up to 10 people) often do not have the knowledge and capability to do all the tasks we are required to do if we are to design a fully automated delivery or deployment pipeline. If you are a member of a team in charge of a development of a (micro)service, chances are you are not an expert in continuous delivery or deployment (CD) tools, you might not be an expert in clusters, networking, and so on. On the other hand, you do not want to deliver your automated tests in such a state that someone else would need to spend endless hours trying to figure out which dependencies they need, how they should be run, what is a definition of success, and so on.

In this article I'll propose a simple way to remove dependencies between different levels of expertise without sacrificing anything. We'll see an example of the way we can define our tests using only Docker. We'll use the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project as example and explore how it is tested. I will assume that you have, at least basic, knowledge of how [Docker Engine](https://www.docker.com/products/docker-engine) and [Docker Compose](https://www.docker.com/products/docker-compose) work. Since the aim is simplicity, this article will be agnostic to CD tools and you should be able to apply the knowledge to any of them.

Let's start with testing requirements.

Testing Requirements
--------------------

First of all, testing should not depend on any specific infrastructure, dependencies, nor tools. You should be able to run them no matter whether you're using Jenkins or Team City, whether your infrastructure is running Ubuntu or RedHat, or whether you are writing services in NodeJS, Java, Scala, or Go. The exception to this rule is the need to have Docker Machine running on your servers. Besides that, everyone should be able to run the tests whenever and wherever. To prove the point, I will not expect you to have anything (besides Docker) running on your machine. I won't even discuss which programming language *Docker Flow: Proxy* uses, which libraries it needs, nor how it compiles. Those things should not be important to anyone but people who wrote the code. True automation with Docker should be agnostic to those details.

The second requirement is to have clearly defined type of tests. For the sake of this article, we'll have three stages: *unit*, *staging*, and *production*. You might have different types in your organization and the aim of this article is not to argue for one over the other. For such a discussion, please visit the [Continuous Deployment with Containers](https://www.infoq.com/articles/continuous-deployment-containers) article.

In the *Docker Flow: Proxy* case, *unit* tests are all the tests that do not require any external dependency. Those are the tests that do not require the service to be compiled and has all dependencies mocked. The same process compiles the service if all tests passed.

The tests that belong to the *staging* group require the service to be running together with all the dependencies. It is as close to production environment as we can be.

Finally, the tests run in the *production* phase are those that are run after the service is deployed to production. They should confirm that deployment was indeed performed correctly. Later on, I'll argue that those tests should be run twice in the context of blue-green deployment.

The last requirement is more an assumption. I assume that the whole organization is capable of coming up with a simple set of conventions. If should agree that, for example, staging tests should always have the same name of the target. That way, we can reuse the same deployment pipeline no matter whether we have ten, hundred, or thousand projects.

In this article, the conventions will be as follows.

Relevant Docker Compose targets are called *unit*, *staging*, and *production*. Definitions are divided into *docker-compose.yml* and *docker-compose-test.yml*. The first one is meant to production deployment while the later should be used for all types of testing. We'll also assume that the service is defined as target *app* inside *docker-compose.yml*. Finally, we'll define *HOST_IP* variable in case someone needs to know the IP of the host where dependencies are running.

Let's give it a spin and run a few examples. Later on we'll discuss them in more detail.

Giving It a Spin
----------------

The examples that follow assume that your laptop is running [Docker Engine](https://www.docker.com/products/docker-engine), [Docker Compose](https://www.docker.com/products/docker-compose), [Docker Machine](https://www.docker.com/products/docker-machine), and [VirtualBox](https://www.virtualbox.org/). Please install them if you don't have them already.

We'll start by pulling the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) project from GitHub.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```

Please note that Docker Machine shares your host users directory with the virtual machine it created. Make sure that you cloned the code inside one of its subdirectories. In my case, on *OS X*, the code was checked out inside the */Users/vfarcic/projects/docker-flow-proxy* directory.

We'll create a new Docker Machine that will be used throughout this article. Feel free to skip this step if you are running Docker natively.

```bash
docker-machine create -d virtualbox docker-flow-proxy
```

At the end of the command output you'll see instructions how to connect to the Docker Client running inside the machine we created. In case of Linux and OS X distributions, the commands are as follows.

```bash
eval $(docker-machine env docker-flow-proxy)

export HOST_IP=$(docker-machine ip docker-flow-proxy)
```

Now we are ready to run tests.

Since we already agreed that the first set of tests should be called *unit* and we assume that their definition is inside *docker-compose-test.yml*, we can run them without any additional discussion.

```bash
docker-compose -f docker-compose-test.yml run --rm unit
```

The tail of output of the `docker-compose` command is as follows.

```
PASS
coverage: 97.8% of statements
```

All the tests passed and we are ready to move onto the next stage. Please note that, besides running the tests, it compiled the code into a binary *docker-flow-proxy*.

Since we used the `--rm` argument, Docker removed the container when it finished the execution. We can confirm that by running `docker ps` command.

```bash
docker ps -a
```

Now we can build the service container. Again, we use conventions and assume that the service is defined inside *docker-compose.yml* (default compose path) as target *app*.

```bash
docker-compose build app
```

The `build` command should be followed with `push` but we'll skip it, since such an action would require that I make my Docker Hub username and password public. Just imagine that we pushed it to the registry.

Let's proceed with *staging* tests. This one is a bit more complicated since the tests require some dependencies. Like before, we don't care what those dependencies are. We'll just assume that targets *staging-dep* and *staging* are defined inside *docker-compose-test.yml*. The *staging-dep* target will run all the dependencies while *staging* will run the tests that use those dependencies.

This time, the process will be a bit more complicated. We need to run the dependencies, followed with tests. Once we're done, we should remove all the containers so that the next que can safely repeat the same process.

```bash
docker-compose -f docker-compose-test.yml up -d staging-dep

docker-compose -f docker-compose-test.yml run --rm staging

docker-compose -f docker-compose-test.yml down
```

The first command (`up -d staging-dep`) run all the dependencies, the second (`run --rm staging`) run the tests, and the last one (`down`) removed all the containers from the system.

This is the moment when we can proceed with deployment to production. I'll skip those steps since you probably got the point what we're trying to accomplish. I'll just say that, after pushing the new container to the registry, we'd deploy it to production and continue using the conventions. This time we'd run `docker-compose -f docker-compose-test.yml run --rm production`.

As you can see, we run all the tests without having any knowledge of what is inside those tests, how the service should be built, what language was used to code it, or which dependencies are needed.

Conventions, conventions, conventions
-------------------------------------

The whole process is based on assumptions and conventions. We assume that everything is running as containers and made the following conventions.

* All tests are defined in *docker-compose-test.yml* file.
* Service itself is defined in *docker-compose.yml* file.
* The *unit* target runs tests that do not require the service to be deployed and, if needed, builds binaries.
* The *staging* target runs tests in staging environment. Since those tests do require the rest of the system to be up and running, it assumes that *staging-dep* target is defined as well.
* The *production* target runs tests that assume that the service is deployed to production. Unlike *staging*, it does not run any dependencies since it assumes that production is fully set up.
* In case any of those tests require the knowledge of the host, the *HOST_IP* environment variable is defined as well.

Thanks to Docker and a very simple set of conventions, we do not need to bang our heads over dependencies and we can stop building deployment pipelines that differ from one project to another. The same process and the set of commands behind it can be reused across the whole organization. No matter whether we have ten or a thousands services, all deployment pipelines will be exactly the same reducing CD maintenance a breeze. The work of defining all those targets is moved from infrastructure teams (or whomever was in charge of automation) to developers. Each team can maintain their own Docker Compose files and assume that their commits to the code repository will pass the whole pipeline.

The end result is deployment pipeline defined as code and stored in the same repository as the code. Whichever CD tool we use will be able to have exactly the same definition without loosing the flexibility to develop our services in any way we find fit.

Let's take a look at the Compose files in more detail.

Looking Inside the Box
----------------------

All those assumptions and conventions would be worthless if we do not define Docker Compose targets. The aim of the description that follows should serve only as guidelines that should be adapted to your own needs.

The *Docker Flow: Proxy* service is written in [Go](https://golang.org/). Therefore, running unit tests requires Go compiler. Chances are that you do not have it installed in your laptop and, even if you do, the Docker Machine we created is "Go free". That is in line with one of the assumptions that everything is running inside containers. As a result, OS running tests does not require anything but Docker Engine and Docker Compose.

The *unit* target inside the [docker-compose-test.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-test.yml) is as follows.

```
  unit:
    image: golang:1.6
    volumes:
      - .:/usr/src/myapp
      - /tmp/go:/go
    command: bash -c "cd /usr/src/myapp && go get -d -v -t && go test --cover -v ./... && go build -v -o docker-flow-proxy"
```

It uses `golang` image that provides the Go compiler and runtimes and it exposes few volumes that are shared between the host and the container. The first volume is the current directory (*.*) that contains the source code. The second is not mandatory but helps speeding up the process. It shares the directory with project dependencies (go libraries I use) between the host and the container. That way, they are downloaded once and uses across multiple tests executions. Finally, the `command` executes commands that download the Go libraries (`go get -d -v -t`), runs tests (`go test --cover -v ./...`), and builds the binary (`go build -v -o docker-flow-proxy`).

The process would be similar no matter which language the service uses. Use the appropriate image (e.g. Java, Scala, NodeJS, and so on), share few directories, execute the command that runs the tests, compiles, creates documentation, and so on. In other words, specify as `command` everything that needs to happen before you start building the service container that will, ultimately, be deployed to production.

Next in line is *staging*. As you've seen, it consists of two targets: *staging-dep* and *staging*.

The *staging-dep* targets runs all the dependent services. The target defined in [docker-compose-test.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-test.yml) is as follows.

```
  staging-dep:
    image: vfarcic/docker-flow-proxy
    environment:
      CONSUL_ADDRESS: ${HOST_IP}:8500
    ports:
      - 80:80
      - 8080:8080
    depends_on:
      - consul
      - registrator
      - app-1
      - app-2
      - app-3
```

This targets launches the service (`vfarcic/docker-flow-proxy`) so that it can be tested. However, both the service and the tests need a few other dependencies, we need to run them as well. The are specified inside the `depends_on` section. Please consult [docker-compose-test.yml](https://github.com/vfarcic/docker-flow-proxy/blob/master/docker-compose-test.yml) for the details behind those dependencies.

Once we run those dependencies, we can execute the tests defined inside the `staging` target. Its definition is as follows.

```
  staging:
    image: golang:1.6
    volumes:
      - .:/usr/src/myapp
      - /tmp/go:/go
    environment:
      - DOCKER_IP=${HOST_IP}
      - CONSUL_IP=${HOST_IP}
    command: bash -c "cd /usr/src/myapp && go get -d -v -t && go test --tags integration"
```

This one is similar to the `unit` target. It uses the `golang:1.6`, shares few volumes, defines few environment variables, and runs the required commands.

The `production` target is as follows.

```
  production:
    extends:
      service: staging
```

Since, in this case, production tests are the same as those defined in the `staging` target, `production`, simply, extends it. While we could run `staging` target in production and get rid of the `production` target, doing so would brake the convention that requires the existence of the `production` target. We had to write a few additional lines but, in my opinion, that cost is smaller than if we had to brake the assumption causing the pipeline to treat this service differently. As the additional benefit, if, in the future, we decide to run a different set of tests in production or use different parameters, we'd only have to modify this target without changing the deployment flow.

What's Next
-----------

In the next article we'll go deeper into the subject and explore how to incorporate those tests into a fully operating deployment pipeline implemented with Jenkins.

Before you leave, please destroy the machine we created. You might need the resources it uses for something else.

```bash
docker-machine rm -f docker-flow-proxy
```

If you are looking for inspiration, please check out the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) as well as the parent project [Docker Flow](https://github.com/vfarcic/docker-flow). Please let me know if you find them useful or you're in need of help.

The DevOps 2.0 Toolkit
----------------------

<a href="http://www.amazon.com/dp/B01BJ4V66M" rel="attachment wp-att-3017"><img src="https://technologyconversations.files.wordpress.com/2014/04/the-devops-2-0-toolkit.png?w=188" alt="The DevOps 2.0 Toolkit" width="188" height="300" class="alignright size-medium wp-image-3017" /></a>If you liked this article, you might be interested in [The DevOps 2.0 Toolkit: Automating the Continuous Deployment Pipeline with Containerized Microservices](http://www.amazon.com/dp/B01BJ4V66M) book. Among many other subjects, it explores *Docker*, *testing* and *continuous deployment pipelines* in much more detail.

The book is about different techniques that help us architect software in a better and more efficient way with *microservices* packed as *immutable containers*, *tested* and *deployed continuously* to servers that are *automatically provisioned* with *configuration management* tools. It's about fast, reliable and continuous deployments with *zero-downtime* and ability to *roll-back*. It's about *scaling* to any number of servers, the design of *self-healing systems* capable of recuperation from both hardware and software failures and about *centralized logging and monitoring* of the cluster.

In other words, this book envelops the whole *microservices development and deployment lifecycle* using some of the latest and greatest practices and tools. We'll use *Docker, Ansible, Ubuntu, Docker Swarm and Docker Compose, Consul, etcd, Registrator, confd, Jenkins, nginx*, and so on. We'll go through many practices and, even more, tools.

The book is available from Amazon ([Amazon.com](http://www.amazon.com/dp/B01BJ4V66M) and other worldwide sites) and [LeanPub](https://leanpub.com/the-devops-2-toolkit).

TODO
----

* Test in Windows
* *Docker Flow: Proxy* marketing.