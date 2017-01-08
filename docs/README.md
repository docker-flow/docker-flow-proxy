```bash
docker-compose -f docker-compose-test.yml run --rm docs

docker build \
    -t vfarcic/docker-flow-proxy-docs \
    -f Dockerfile.docs \
    .

docker push vfarcic/docker-flow-proxy-docs
```