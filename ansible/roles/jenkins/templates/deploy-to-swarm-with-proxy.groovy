node("docker") {
    git "https://github.com/vfarcic/docker-flow-proxy.git"
    withEnv(["DOCKER_HOST=tcp://10.100.192.200:2375"]) {
        sh "docker-compose -p books-ms -f docker-compose-demo.yml up -d"
    }
    sh "curl \"proxy:8081/v1/docker-flow-proxy/reconfigure?serviceName=books-ms&amp;servicePath=/api/v1/books\""
}