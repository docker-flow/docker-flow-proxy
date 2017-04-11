## Environment variables

```bash
export DIGITALOCEAN_ACCESS_TOKEN=[...]

export DIGITALOCEAN_API_TOKEN=[...]

export DIGITALOCEAN_TOKEN=[...]

export DIGITALOCEAN_REGION=sfo1

ssh-keygen -t rsa # proxy-key
```

## Snapshot

```bash
packer build -machine-readable \
    proxy.json \
    | tee proxy.log

export TF_VAR_swarm_snapshot_id=$(\
    grep 'artifact,0,id' \
    proxy.log \
    | cut -d: -f2)
```

## New cluster

```bash
terraform plan

terraform apply \
    -target digitalocean_droplet.swarm-manager-1 \
    -var swarm_init=true

export TF_VAR_swarm_manager_token=$(ssh \
    -i proxy-key \
    root@$(terraform output \
    swarm_manager_1_public_ip) \
    docker swarm join-token -q manager)

export TF_VAR_swarm_worker_token=$(ssh \
    -i proxy-key \
    root@$(terraform output \
    swarm_manager_1_public_ip) \
    docker swarm join-token -q worker)

export TF_VAR_swarm_manager_ip=$(terraform \
    output swarm_manager_1_private_ip)

terraform apply
```

## Cluster update

```bash
# Add a new manager node to the cluster

# Start replacing nodes one by one
```

## SSH

```bash
ssh -i proxy-key \
    root@$(terraform \
    output swarm_manager_1_public_ip)
```

## Services

```bash
docker network create --driver overlay proxy

curl -o proxy.yml \
    https://raw.githubusercontent.com/vfarcic/\
docker-flow-proxy/master/do/stack.yml

docker stack deploy -c proxy.yml proxy

curl -o swarm-listener.yml \
    https://raw.githubusercontent.com/vfarcic/\
docker-flow-swarm-listener/master/stack.yml

docker stack deploy -c swarm-listener.yml swarm-listener
```