```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

docker service create \
    --name=sample-ci-01 \
    --network=proxy \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.port.1=8080 \
    --label com.df.servicePath.1=/jenkins-1 \
    --label com.df.reqMode.1=http \
    --label com.df.srcPort.2=50000 \
    --label com.df.port.2=50000 \
    --label com.df.reqMode.2=tcp \
    --label com.df.servicePath.2=/ \
    -e JENKINS_OPTS="--prefix=/jenkins-1" \
    --detach=true \
    jenkinsci/jenkins:lts-alpine

# NOTE: Removed --constraint 'node.role==worker'
# NOTE: Removed --label com.df.serviceDomain=sample.ci.mycompany.local
# NOTE: Removed --env "GIT_SEED=sample-applications"
# NOTE: --label com.df.servicePath.1=/jenkins-1 # Changed to `/jenkins-1`
# NOTE: Added `reqMode`

export HOST_IP=[...] # Change to what is the IP of your host

docker service logs sample-ci-01

# NOTE: Copy the initial password

open "http://$HOST_IP/jenkins-1"

# NOTE: Use `admin` as both username and password

open "http://$HOST_IP/jenkins-1/configure"

# NOTE: Confirm that `Jenkins URL` is correct

# NOTE: Click the `Save` button (even if everything is OK)

open "http://$HOST_IP/jenkins-1/pluginManager/available"

# Install "Self-Organizing Swarm Plug-in Modules" plugin

docker service create \
    --name sample-ci-01-agent \
    -e "COMMAND_OPTIONS=-master http://${HOST_IP}/jenkins-1 -username admin -password admin -executors 1" \
    --mount "type=bind,source=/var/run/docker.sock,destination=/var/run/docker.sock" \
    vfarcic/jenkins-swarm-agent


```