```bash
# https://serversforhackers.com/using-ssl-certificates-with-haproxy

mkdir -p tmp

openssl genrsa -out tmp/xip.io.key 1024

openssl req -new \
    -key tmp/xip.io.key \
    -out tmp/xip.io.csr

# ES
# Barcelona
# Barcelona
# TechnologyConversations.com
#
# *.xip.io
# viktor@farcic.com
#
#

openssl x509 -req -days 365 \
    -in tmp/xip.io.csr \
    -signkey tmp/xip.io.key \
    -out tmp/xip.io.crt

cat tmp/xip.io.crt tmp/xip.io.key \
    | tee tmp/xip.io.pem

docker network create --driver overlay proxy

docker network create --driver overlay go-demo

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    --replicas 3 \
    --network proxy \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy

docker service create --name go-demo-db \
    --network go-demo \
    mongo

docker service create --name go-demo \
    -e DB=go-demo-db \
    --network go-demo \
    --network proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/demo \
    --label com.df.port=8080 \
    --label com.df.serviceDomain=xip.io \
    vfarcic/go-demo

docker service ls
```

```
ID            NAME            REPLICAS  IMAGE                               COMMAND
5jhdwyrcin5c  go-demo         1/1       vfarcic/go-demo
79g8pomh4lj8  go-demo-db      1/1       mongo
be8pgp5fo6aw  proxy           3/3       vfarcic/docker-flow-proxy
da8o3wje0r3e  swarm-listener  1/1       vfarcic/docker-flow-swarm-listener
```

```bash
# Find out the IP

IP=192.168.1.34

curl -i http://$IP.xip.io/demo/hello
```

```
HTTP/1.1 200 OK
Date: Sat, 12 Nov 2016 22:51:18 GMT
Content-Length: 14
Content-Type: text/plain; charset=utf-8

hello, world!
```

```bash
curl -i https://$IP.xip.io/demo/hello
```

```
curl: (35) Unknown SSL protocol error in connection to 192.168.1.34.xip.io:-9847
```

```bash
# TODO: Remove

docker service update \
    --publish-add 8080:8080 proxy

docker service ps proxy

# TODO: Wait until it's running
```

```
ID                         NAME         IMAGE                      NODE  DESIRED STATE  CURRENT STATE            ERROR
4nm8r1vch5dalyeenvxx4nfhm  proxy.1      vfarcic/docker-flow-proxy  moby  Running        Running 8 seconds ago
eqh2mes1wxfaas1xpohkv5oj4   \_ proxy.1  vfarcic/docker-flow-proxy  moby  Shutdown       Shutdown 11 seconds ago
a2rinfst58xb5h7uk27p3upk6  proxy.2      vfarcic/docker-flow-proxy  moby  Running        Running 18 seconds ago
8h5qhvlmcstjgf0zm70qpl1lz   \_ proxy.2  vfarcic/docker-flow-proxy  moby  Shutdown       Shutdown 21 seconds ago
cqzjsej3nje3lhuy9ovwfgnb6  proxy.3      vfarcic/docker-flow-proxy  moby  Running        Running 13 seconds ago
a2la3xr7r8u3341ryczt2ln7d   \_ proxy.3  vfarcic/docker-flow-proxy  moby  Shutdown       Shutdown 16 seconds ago
```


```bash
# TODO: Remove

docker service inspect proxy --pretty
```

```
ID:		be8pgp5fo6awkinrdf93xfsun
Name:		proxy
Mode:		Replicated
 Replicas:	3
Update status:
 State:		completed
 Started:	3 minutes ago
 Completed:	3 minutes ago
 Message:	update completed
Placement:
UpdateConfig:
 Parallelism:	1
 On failure:	pause
ContainerSpec:
 Image:		vfarcic/docker-flow-proxy
 Env:		LISTENER_ADDRESS=swarm-listener MODE=swarm
Resources:
Networks: dsqqdv2cmqyongmno9wytfnzg
Ports:
 Protocol = tcp
 TargetPort = 443
 PublishedPort = 443
 Protocol = tcp
 TargetPort = 80
 PublishedPort = 80
 Protocol = tcp
 TargetPort = 8080
 PublishedPort = 8080
```

```bash
curl localhost:8080/v1/docker-flow-proxy/config
```

```
global
    pidfile /var/run/haproxy.pid

defaults
    mode    http
    balance roundrobin

    option  dontlognull
    option  dontlog-normal
    option  http-server-close
    option  forwardfor
    option  redispatch

    maxconn 5000
    timeout connect 5s
    timeout client  20s
    timeout server  20s
    timeout queue   30s
    timeout http-request 5s
    timeout http-keep-alive 15s

    stats enable
    stats refresh 30s
    stats realm Strictly\ Private
    stats auth admin:admin
    stats uri /admin?stats

frontend services
    bind *:80
    bind *:443


    acl url_go-demo path_beg /demo
    acl domain_go-demo hdr_dom(host) -i xip.io
    use_backend go-demo-be if url_go-demo domain_go-demo

backend go-demo-be
    server go-demo go-demo:8080
```

```bash
curl -i -XPUT \
    --data-binary @tmp/xip.io.pem \
    "localhost:8080/v1/docker-flow-proxy/cert?certName=xip.io.pem&distribute=true"
```

```
HTTP/1.1 100 Continue

HTTP/1.1 200 OK
Content-Type: application/json
Date: Sat, 12 Nov 2016 23:04:39 GMT
Content-Length: 28

{"Status":"OK","Message":""}
```

```bash
curl localhost:8080/v1/docker-flow-proxy/config

# TODO: Note that the certificated have been distributed to all instances

curl -i \
    $(docker-machine ip docker-flow-proxy-tests):8080/v1/docker-flow-proxy/certs

# TODO: Show that certs are distributed

# TODO: Show that certs are recuperated from existing instances on scale
```