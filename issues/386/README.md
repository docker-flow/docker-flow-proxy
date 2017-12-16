```bash
# Compile

docker container run --rm -it -v $PWD:/src vfarcic/gox docker-flow-proxy

# Release

export GITHUB_TOKEN=[...]

docker container run --rm -it -e GITHUB_TOKEN=$GITHUB_TOKEN -v $PWD:/src -w /src vfarcic/github-release

#msg = sh(returnStdout: true, script: "git log --format=%B -1").trim()

# Change -a and -m
docker container run --rm -it -e GITHUB_TOKEN=$GITHUB_TOKEN -v $PWD:/src -w /src vfarcic/github-release git tag -a testing2 -m "Testing automated releases"

docker container run --rm -it -e GITHUB_TOKEN=$GITHUB_TOKEN -v $PWD:/src -w /src vfarcic/github-release git push --tags

# Change --repo, --tag, --name, --description
docker container run --rm -it -e GITHUB_TOKEN=$GITHUB_TOKEN -v $PWD:/src -w /src vfarcic/github-release github-release release --user vfarcic --repo docker-flow-proxy --tag testing --name "Testing name" --description "Testing description"

# Change --repo, --tag, --name, --file
docker container run --rm -it -e GITHUB_TOKEN=$GITHUB_TOKEN -v $PWD:/src -w /src vfarcic/github-release github-release upload --user vfarcic --repo docker-flow-proxy --tag testing --name "docker-flow-proxy_darwin_386" --file docker-flow-proxy_darwin_386







node("prod") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    msg = sh(returnStdout: true, script: "git log --format=%B -1").trim()
    println msg
    println "---"
    files = findFiles(glob: '*')
    println files
    println "---"
    for (def file : files) {
        println file
    }
}






$GOPATH/bin/github-release delete \
    --user vfarcic \
    --repo docker-flow-proxy \
    --tag testing
```