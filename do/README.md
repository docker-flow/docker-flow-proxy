## Environment variables

```bash
export DIGITALOCEAN_ACCESS_TOKEN=[...]

export DIGITALOCEAN_API_TOKEN=[...]

export DIGITALOCEAN_TOKEN=[...]

export DIGITALOCEAN_REGION=sfo2

ssh-keygen -t rsa # proxy-key
```

## Snapshot

```bash
packer build -machine-readable \
    proxy.json \
    | tee proxy.log
```

## Swarm Instances

```bash
export TF_VAR_swarm_snapshot_id=$(\
    grep 'artifact,0,id' \
    proxy.log \
    | cut -d, -f6 | cut -d: -f2)

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
docker-flow-proxy/master/docker-compose-stack.yml

docker stack deploy -c proxy.yml proxy

docker service create --name proxy-docs \
    --network proxy \
    --replicas 2 \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/ \
    --label com.df.port=80 \
    vfarcic/docker-flow-proxy-docs

docker service create --name letsencrypt-companion \
    --label com.df.notify=true \
    --label com.df.distribute=true \
    --label com.df.servicePath=/.well-known/acme-challenge \
    --label com.df.port=80 \
    -e DOMAIN_1="('dockerflow.com' 'www.dockerflow.com' 'proxy.dockerflow.com' 'registry.dockerflow.com')" \
    -e DOMAIN_COUNT=1 \
    -e CERTBOT_EMAIL="viktor@farcic.com" \
    -e PROXY_ADDRESS="proxy" \
    -e CERTBOT_CRON_RENEW="('0 3 * * *' '0 15 * * *')" \
    --network proxy \
    --mount type=bind,source=/etc/letsencrypt,destination=/etc/letsencrypt \
    hamburml/docker-flow-letsencrypt

docker service create --name registry \
    --label com.df.notify=true \
    --label com.df.servicePath=/ \
    --label com.df.serviceDomain=registry.dockerflow.com \
    --label com.df.distribute=true \
    --label com.df.port=5000 \
    --network proxy \
    registry:2

curl -o logging.yml \
    https://raw.githubusercontent.com/vfarcic/\
docker-flow-stacks/master/logging/logging-df-proxy.yml

docker stack deploy -c logging.yml logging

curl -o metrics.yml \
    https://raw.githubusercontent.com/vfarcic/\
docker-flow-stacks/master/metrics/prometheus-grafana-df-proxy.yml

export DOMAIN=grafana.dockerflow.com

docker stack deploy -c metrics.yml metrics

exit

open "http://proxy.dockerflow.com"
```