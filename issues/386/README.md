```bash
# Change --repo, --tag, --name, --file







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