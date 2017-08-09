```
docker network create -d overlay proxy

docker network create -d overlay monitor

TAG=beta docker stack deploy -c stack.yml proxy

docker stack ps proxy

curl "http://localhost/demo/hello"

curl "http://localhost:8080/metrics"

open "http://localhost/monitor/targets"

for ((n=0;n<100;n++)); do
    curl "http://localhost/demo/hello"
done

open "http://localhost/monitor/graph"

# haproxy_frontend_http_requests_total
```
