```bash
docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    -e DFP_SERVICE_1_SERVICE_NAME=go-demo_main \
    -e DFP_SERVICE_1_SERVICE_PATH=/demo \
    -e DFP_SERVICE_1_PORT=8080 \
    vfarcic/docker-flow-proxy
```