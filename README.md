Docker Flow: Proxy
==================

The goal of the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) project is to provide an easy way to reconfigure proxy every time a new service is deployed or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and join them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and [Consul](https://www.consul.io/) for service discovery. On top of those two, it adds custom logic that allows on-demand reconfiguration of the proxy.

For a practical example please read the [TODO](TOD) article.

Arguments
---------

### Environment Variables

The following

Arguments can be specified through *docker-flow.yml* file, environment variables, and command line arguments. If the same argument is specified in several places, command line overwrites all others and environment variables overwrite *docker-flow.yml*.

### Command line arguments

|Command argument       |Environment variable  |YML              |Description|
|-----------------------|----------------------|-----------------|-----------|
|-F, --flow             |FLOW                  |flow             |The actions that should be performed as the flow. (**multi**)<br>**deploy**: Deploys a new release<br>**scale**: Scales currently running release<br>**stop-old**: Stops the old release<br>**proxy**: Reconfigures the proxy<br>(**default**: [deploy])|
|-H, --host             |FLOW_HOST             |host             |Docker daemon socket to connect to. If not specified, DOCKER_HOST environment variable will be used instead.|
|-f, --compose-path     |FLOW_COMPOSE_PATH     |compose_path     |Path to the Docker Compose configuration file. (**default**: docker-compose.yml)|
|-b, --blue-green       |FLOW_BLUE_GREEN       |blue_green       |Perform blue-green deployment. (**bool**)|
|-t, --target           |FLOW_TARGET           |target           |Docker Compose target. (**required**)|
|-T, --side-target      |FLOW_SIDE_TARGETS     |side_targets     |Side or auxiliary Docker Compose targets. (**multi**)|
|-P, --skip-pull-targets|FLOW_SKIP_PULL_TARGET |skip_pull_target |Skip pulling targets. (**bool**)|
|-S, --pull-side-targets|FLOW_PULL_SIDE_TARGETS|pull_side_targets|Pull side or auxiliary targets. (**bool**)|
|-p, --project          |FLOW_PROJECT          |project          |Docker Compose project. If not specified, the current directory will be used instead.|
|-c, --consul-address   |FLOW_CONSUL_ADDRESS   |consul_address   |The address of the Consul server. (**required**)|
|-s, --scale            |FLOW_SCALE            |scale            |Number of instances to deploy. If the value starts with the plus sign (+), the number of instances will be increased by the given number. If the value begins with the minus sign (-), the number of instances will be decreased by the given number.|
|-r, --proxy-host       |FLOW_PROXY_HOST       |proxy_host       |Docker daemon socket of the proxy host.|

Arguments can be strings, boolean, or multiple values. Command line arguments of boolean type do not have any value (i.e. *--blue-green*). Environment variables and YML arguments of boolean type should use *true* as value (i.e. *FLOW_BLUE_GREEN=true* and *blue_green: true*). When allowed, multiple values can be specified by repeating the command line argument (i.e. *--flow=deploy --flow=stop-old*). When specified through environment variables, multiple values should be separated by comma (i.e. *FLOW=deploy,stop-old*). YML accepts multiple values through the standard format.

```yml
flow:
  - deploy
  - stop-old
```

TODO
----

* docker-flow

  * Add proxy to the code
  * Add proxy to the README

* Add description to Docker Hub
* Article

  * Proofread
  * Copy to README
  * Reference README, docker-flow README, and docker-flow article.
  * Publish

* New Docker Flow article with blue-green, scaling, and proxy

  * Write
  * Proofread
  * Publish
  * Add reference to README
