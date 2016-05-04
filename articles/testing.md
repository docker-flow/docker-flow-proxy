I was involved in many automation projects and come to realize that the key aspect is a clear separation of concerns. This time, by separation of concerns, I'm not referring to design principle for separating a computer program into distinct sections but separation based on different roles people have within organization they work in as well as tasks they perform. This was never so true as when we adopt microservices architecture and containers. There is no real reason, any more, not to let people inside the project be in charge of the whole deployment pipeline. When we start doing hand-overs to others (people not working with us on day-to-day basis) we start having delays. The problem is that small teams (up to 10 people) often do not have the knowledge and capability to do all the tasks we are required to do if we are to design a fully automated delivery or deployment pipeline. If you are a member of a team in charge of a development of a (micro)service, chances are you are not an expert in continuous delivery or deployment (CD) tools, you might not be an expert in clusters, networking, and so on. On the other hand, you do not want to deliver your automated tests in such a state that someone else would need to spend endless our trying to figure out which dependencies they need, how they should be run, what is a defition of success, and so on.

In this article I'll propose a simple way to remove dependencies between different levels of expertise without sacrificing anything. We'll see an example of the way we can define our tests using Docker, Jenkins, and few other tools. We'll use the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project as example and explore how it is tested. I will assume that you have, at least basic, knowledge of how [Docker Engine](https://www.docker.com/products/docker-engine) and [Docker Compose](https://www.docker.com/products/docker-compose) work. Since the aim is simplicity, this article will be agnostic to CD tools and you should be able to apply the knowledge to any of them.

Let's start with testing requirements.

Testing Requirements
--------------------

First of all, testing should not depend on any specific infrastructure, dependencies, nor tools. You should be able to run them no matter whether you're using Jenkins or Team City, whether your infrastructure is running Ubuntu or RedHat, or whether you are writing services in NodeJS, Java, Scala, or Go. The exception to this rule is the need to have Docker Machine running on your servers. Besides that, everyone should be able to run the tests whenever and wherever. To prove the point, I will not expect you to have anything (besides Docker) running on your machine. I won't even discuss which programming language *Docker Flow: Proxy* uses, which libraries it needs, nor how it compiles. Those things should not be important to anyone but people who wrote the code. True automation with Docker should be agnostic to those details.

The second requirement is to have clearly defined type of tests. For the sake of this article, we'll have three stages: *unit*, *staging*, and *production*. You might have different types in your organization and the aim of this article is not to argue for one over the other. For such a discussion, please visit the [Continuous Deployment with Containers](https://www.infoq.com/articles/continuous-deployment-containers) article.

In the *Docker Flow: Proxy* case, *unit* tests are all the tests that do not require any external dependency. Those are the tests that do not require the service to be compiled and has all dependencies mocked. The same process compiles the service if all tests passed.

The tests that belong to the *staging* group require the service to be running together with all the dependencies. It is as close to production environment as we can be.

Finally, the tests run in the *production* phase are those that are run after the service is deployed to production. They should confirm that deployment was indeed performed correctly. Later on, I'll argue that those tests should be run twice in the context of blue-green deployment.

The last requirement is more an assumption. I assume that the whole organization is capable of coming up with a simple set of conventions. If should agree that, for example, staging tests should always have the same name of the target. That way, we can reuse the same deployment pipeline no matter whether we have ten, hundred, or thousand projects.

Let's give it a spin and run a few examples. Later on we'll discuss them in more detail.

Giving It a Spin
----------------

The examples that follow assume that your laptop is running [Docker Engine](https://www.docker.com/products/docker-engine), [Docker Compose](https://www.docker.com/products/docker-compose), and [Docker Machine](https://www.docker.com/products/docker-machine). Please install them if you don't have them already.

We'll start by pulling the [vfarcic/docker-flow-proxy](https://github.com/vfarcic/docker-flow-proxy) project from GitHub.

```bash
git clone https://github.com/vfarcic/docker-flow-proxy.git

cd docker-flow-proxy
```