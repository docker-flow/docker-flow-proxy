```bash
vagrant up

vagrant ssh test

sudo yum install -y yum-utils device-mapper-persistent-data lvm2

sudo yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo

sudo yum makecache fast

sudo yum install -y docker-ce

sudo mkdir -p /etc/docker/

echo '{
  "storage-driver": "devicemapper"
}' | sudo tee /etc/docker/daemon.json

sudo systemctl start docker

sudo docker version

sudo docker swarm init

sudo docker network create --attachable --driver overlay proxy

sudo docker network create --attachable --driver overlay ingenium

sudo docker service create --name swarm-listener \
  --network proxy \
  --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
  -e DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
  -e DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
  vfarcic/docker-flow-swarm-listener

sudo docker service create --name proxy \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  --network proxy \
  -e MODE=swarm \
  -e LISTENER_ADDRESS=swarm-listener \
  vfarcic/docker-flow-proxy:1.402

sudo docker service create --name go-demo-db \
  --network ingenium mongo

sudo docker service create --name go-demo \
  -e DB=go-demo-db \
  --network ingenium \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  vfarcic/go-demo

sudo docker service ls # Wait until everything is running

curl "http://0.0.0.0/demo/hello"

sudo docker container logs \
  $(sudo docker container ls -q \
  -f "label=com.docker.swarm.service.name=proxy")

# Outputs Proxy config was reloaded

sudo docker service rm go-demo

sudo docker container logs \
  $(sudo docker container ls -q \
  -f "label=com.docker.swarm.service.name=proxy")

# Outputs Removing go-demo configuration

sudo docker service create --name go-demo \
  -e DB=go-demo-db \
  --network ingenium \
  --network proxy \
  --label com.df.notify=true \
  --label com.df.distribute=true \
  --label com.df.servicePath=/demo \
  --label com.df.port=8080 \
  vfarcic/go-demo

sudo docker container logs \
  $(sudo docker container ls -q \
  -f "label=com.docker.swarm.service.name=proxy")

# Outputs Creating configuration for the service go-demo

sudo docker container logs -f \
  $(sudo docker container ls -q \
  -f "label=com.docker.swarm.service.name=swarm-listener")

# It does not continue with the message "Retrying service created notification..."
```