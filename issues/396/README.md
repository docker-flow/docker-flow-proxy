```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

curl -i "http://localhost/demo/hello" # Wait until 200

docker service scale proxy_main=0

docker service logs -f proxy_proxy
```

```
...
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Sending distribution request to http://10.0.0.10:8080/v1/docker-flow-proxy/remove?distribute=false&serviceName=proxy_main
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Processing remove request /v1/docker-flow-proxy/remove
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Removing proxy_main configuration
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Removing the proxy_main configuration files
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Reloading the proxy
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Validating configuration
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | Configuration file is valid
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 13:55:33 Proxy config was reloaded
```

```bash
docker service scale proxy_main=2

curl -i "http://localhost/demo/hello"

docker service scale proxy_main=0

docker service logs -f proxy_proxy
```

```
...
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Sending distribution request to http://10.0.0.10:8080/v1/docker-flow-proxy/remove?distribute=false&serviceName=proxy_main
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Processing remove request /v1/docker-flow-proxy/remove
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Removing proxy_main configuration
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Removing the proxy_main configuration files
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Reloading the proxy
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Validating configuration
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | Configuration file is valid
proxy_proxy.1.mksawvb1tlua@linuxkit-025000000001    | 2017/12/18 14:04:19 Proxy config was reloaded
```

```bash
docker service scale proxy_proxy=0

docker service scale proxy_proxy=1

docker service logs -f proxy_proxy
```