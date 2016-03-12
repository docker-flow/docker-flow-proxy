Setup
===========

```bash
docker-machine create \
    -d virtualbox \
    docker-flow

eval "$(docker-machine env docker-flow)"

export DOCKER_IP=$(docker-machine ip docker-flow)

docker run -d \
    -p "8500:8500" \
    -h "consul" \
    --name "consul" \
    progrium/consul -server -bootstrap

export CONSUL_IP=$(docker-machine ip docker-flow)

docker run -d \
    --name registrator \
    -v /var/run/docker.sock:/tmp/docker.sock \
    -h $DOCKER_IP \
    gliderlabs/registrator \
    -ip $DOCKER_IP consul://$CONSUL_IP:8500
```

HAProxy
=======

```bash
cd haproxy

docker build -t vfarcic/docker-flow-haproxy-consul .

# Start HA

docker run -d \
    --name docker-flow-haproxy-consul \
    -e CONSUL_IP=192.168.99.100 \
    -p 80:80 \
    vfarcic/docker-flow-haproxy-consul

# Start the service

docker-compose up -d app db

# Update HA

docker exec docker-flow-haproxy-consul \
    run.sh books-ms /api/v1/books



docker exec -it docker-flow-haproxy-consul bash











docker run -it --rm -p 8080:80 haproxy bash

apt-get update

apt-get install -y wget unzip

wget https://releases.hashicorp.com/consul-template/0.13.0/consul-template_0.13.0_linux_amd64.zip -O /usr/local/bin/consul-template.zip

unzip /usr/local/bin/consul-template.zip -d /usr/local/bin/

chmod +x /usr/local/bin/consul-template

mkdir -p /usr/local/etc/haproxy/tmpl

# copy haproxy.tmpl and service.ctmpl to /usr/local/etc/haproxy/tmpl
CONSUL_SERVICE=books-ms
CONSUL_URL=/api/v1/books

# Figure out how to fix the case when no services are running
haproxy -f /usr/local/etc/haproxy/haproxy.cfg -D -p /var/run/haproxy.pid

consul-template \
    -consul 192.168.99.100:8500 \
    -template "/usr/local/etc/haproxy/tmpl/service.ctmpl:/usr/local/etc/haproxy/tmpl/books-ms.cfg" \
    -once

cat /usr/local/etc/haproxy/tmpl/haproxy.tmpl /usr/local/etc/haproxy/tmpl/*.cfg >/usr/local/etc/haproxy/haproxy.cfg

haproxy -f /usr/local/etc/haproxy/haproxy.cfg -D -p /var/run/haproxy.pid -sf $(cat /var/run/haproxy.pid)
```

Service
=======


* Tag to +blue/green after pull
* Change image to +blue/green