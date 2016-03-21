node("cd") {
    def prodIp = "10.100.192.200"
    def swarmIp = "10.100.192.200"
    def proxyNode = "swarm-master"
    def swarmPlaybook = "swarm-healing.yml"
    def proxyPlaybook = "swarm-proxy.yml"

    def flow = load "/data/scripts/workflow-util.groovy"
    def currentColor = flow.getCurrentColor(serviceName, prodIp)
    def instances = flow.getInstances(serviceName, swarmIp)

    deleteDir()
    git url: "https://github.com/vfarcic/${serviceName}.git"
    try {
        flow.provision(swarmPlaybook)
        flow.provision(proxyPlaybook)
    } catch (e) {}

    flow.deploySwarm(serviceName, prodIp, currentColor, instances)
    flow.updateBGProxy(serviceName, proxyNode, currentColor)
}