```bash
# Compile

go get github.com/mitchellh/gox

$GOPATH/bin/gox

# Release

go get github.com/aktau/github-release

export GITHUB_TOKEN=[...]

$GOPATH/bin/github-release --help

git tag -a testing -m "Testing automated releases"

git push --tags

$GOPATH/bin/github-release info -u vfarcic -r docker-flow-proxy

$GOPATH/bin/github-release release \
    --user vfarcic \
    --repo docker-flow-proxy \
    --tag testing \
    --name "Testing name" \
    --description "Testing description"

$GOPATH/bin/github-release upload \
    --user vfarcic \
    --repo docker-flow-proxy \
    --tag v0.1.0 \
    --name "gofinance-osx-amd64" \
    --file bin/darwin/amd64/gofinance

$GOPATH/bin/github-release delete \
    --user vfarcic \
    --repo docker-flow-proxy \
    --tag testing
```