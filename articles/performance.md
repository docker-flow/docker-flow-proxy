```bash
docker-machine create \
  --driver virtualbox \
  --virtualbox-boot2docker-url https://github.com/boot2docker/boot2docker/releases/download/v1.13.0-rc4/boot2docker.iso \
  test-routing

eval $(docker-machine env test-routing)

docker swarm init \
    --advertise-addr $(docker-machine ip test-routing):2377

docker network create --driver overlay routing

docker service create --name flow-proxy \
    -p 32503:80 \
    -p 32504:8080 \
    --network routing \
    -e MODE=swarm \
    -e LISTENER_ADDRESS=swarm-listener \
    vfarcic/docker-flow-proxy

docker service create --name swarm-listener \
    --network routing \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://flow-proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://flow-proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role == manager' \
    vfarcic/docker-flow-swarm-listener

docker service create --name nginx \
    --network routing \
    -p 32500:80 \
    --label com.df.notify=true \
    --label com.df.servicePath=/ \
    --label com.df.serviceDomain=nginx.mycluster.com \
    --label com.df.port=80 \
    nginx:1.11.7-alpine

docker service ls

curl -I -H "Host: nginx.mycluster.com" "http://$(docker-machine ip test-routing):32500/"

curl -I -H "Host: nginx.mycluster.com" "http://$(docker-machine ip test-routing):32503/"

ab -n 5000 -c 10 -H "Host: nginx.mycluster.com" "http://$(docker-machine ip test-routing):32500/"
```

```
Server Software:        nginx/1.11.7
Server Hostname:        192.168.99.100
Server Port:            32500

Document Path:          /
Document Length:        612 bytes

Concurrency Level:      10
Time taken for tests:   5.816 seconds
Complete requests:      5000
Failed requests:        0
Total transferred:      4225000 bytes
HTML transferred:       3060000 bytes
Requests per second:    859.74 [#/sec] (mean)
Time per request:       11.631 [ms] (mean)
Time per request:       1.163 [ms] (mean, across all concurrent requests)
Transfer rate:          709.45 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.4      0       7
Processing:     1   11   2.2     11      24
Waiting:        1   11   2.1     10      24
Total:          2   12   2.2     11      26

Percentage of the requests served within a certain time (ms)
  50%     11
  66%     12
  75%     12
  80%     13
  90%     14
  95%     16
  98%     18
  99%     19
 100%     26 (longest request)
```

```bash
ab -n 5000 -c 10 -H "Host: nginx.mycluster.com" "http://$(docker-machine ip test-routing):32503/"
```

```
Server Software:        nginx/1.11.7
Server Hostname:        192.168.99.100
Server Port:            32503

Document Path:          /
Document Length:        612 bytes

Concurrency Level:      10
Time taken for tests:   2.571 seconds
Complete requests:      5000
Failed requests:        0
Total transferred:      4225000 bytes
HTML transferred:       3060000 bytes
Requests per second:    1945.13 [#/sec] (mean)
Time per request:       5.141 [ms] (mean)
Time per request:       0.514 [ms] (mean, across all concurrent requests)
Transfer rate:          1605.11 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.2      0       3
Processing:     2    5   1.5      5      13
Waiting:        2    5   1.5      5      13
Total:          2    5   1.5      5      13

Percentage of the requests served within a certain time (ms)
  50%      5
  66%      6
  75%      6
  80%      6
  90%      7
  95%      8
  98%      9
  99%     10
 100%     13 (longest request)
```

```bash
docker service create --name util \
    --network routing \
    alpine sleep 100000

ID=$(docker ps -q \
    --filter label=com.docker.swarm.service.name=util)

docker exec -it $ID apk add --update apache2-utils

docker exec -it $ID ab -n 5000 -c 10 -H "Host: nginx.mycluster.com" "http://$(docker-machine ip test-routing):32500/"
```

