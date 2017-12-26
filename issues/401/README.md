```bash
docker network create -d overlay proxy

TAG=issue-401 docker stack deploy -c stack.yml proxy

docker service logs proxy_proxy

curl -i -H "Host: rainbowbridge.rip" "http://localhost/demo/hello" # Removed `com.df.srcPort=443`

curl -i -H "Host: www.rainbowbridge.rip" "http://localhost/demo/hello"

curl -i -H "Host: www.rainbowbridge.rip" "https://localhost/demo/hello"
```