```bash
mkdocs build

docker build \
    -t vfarcic/docker-flow-proxy-docs \
    -f Dockerfile.docs \
    .

docker push vfarcic/docker-flow-proxy-docs
```