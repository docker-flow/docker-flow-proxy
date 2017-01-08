# Docker Flow Proxy

The goal of the *Docker Flow Proxy* project is to provide an easy way to reconfigure proxy every time a new service is deployed, or when a service is scaled. It does not try to "reinvent the wheel", but to leverage the existing leaders and combine them through an easy to use integration. It uses [HAProxy](http://www.haproxy.org/) as a proxy and adds custom logic that allows on-demand reconfiguration.

Since the Docker 1.12 release, *Docker Flow Proxy* supports two modes. The default mode is designed to work with any setup and requires Consul and Registrator. The *swarm* mode aims to leverage the benefits that come with *Docker Swarm* and new networking introduced in the 1.12 release. The later mode (*swarm*) does not have any dependency but Docker Engine. The *swarm* mode is recommended for all who use *Docker Swarm* features introduced in v1.12.

Please visit the **[project documentation](proxy.dockerflow.com)** for more info.