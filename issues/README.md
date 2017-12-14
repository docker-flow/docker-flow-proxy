```bash
####################
# Create a cluster #
####################

curl -o swarm-cluster.sh \
    https://raw.githubusercontent.com/vfarcic/docker-flow-proxy/master/scripts/swarm-cluster.sh

chmod +x swarm-cluster.sh

./swarm-cluster.sh

docker-machine ssh node-1

##############
# Deploy DFP #
##############

docker network create --driver overlay proxy

curl -o docker-compose-stack.yml \
    https://raw.githubusercontent.com/\
vfarcic/docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c docker-compose-stack.yml proxy

docker stack ps proxy

# Remember the node of one of the machines where DFP is running (e.g. node-3)

#########################
# Deploy a demo service #
#########################

curl -o docker-compose-go-demo.yml \
    https://raw.githubusercontent.com/\
vfarcic/go-demo/master/docker-compose-stack.yml

docker stack deploy -c docker-compose-go-demo.yml go-demo

docker stack ps go-demo # Wait until all the replicas are running

########
# Test #
########

exit

curl -i "$(docker-machine ip node-1)/demo/hello"

docker-machine rm -f node-3 # Replace node-3 with the node where a DFP replica is running

curl -i "$(docker-machine ip node-1)/demo/hello"

# It might take a while until Swarm detects that a node is down.
# After a while, curl should respond fast.

###########
# Cleanup #
###########

docker-machine rm -f $(docker-machine ls -q)
```