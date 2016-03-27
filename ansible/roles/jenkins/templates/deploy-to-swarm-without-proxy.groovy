node("docker") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    withEnv(["DOCKER_HOST=tcp://10.100.192.200:2375"]) {
        sh "docker-compose -p books-ms -f docker-compose-demo.yml up -d"
    }
}