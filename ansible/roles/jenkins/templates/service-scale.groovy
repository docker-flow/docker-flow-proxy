node("cd") {
    def serviceName = "{{ item.service_name }}"
    def swarmIp = "10.100.192.200"

    def flow = load "/data/scripts/workflow-util.groovy"
    def instances = flow.getInstances(serviceName, swarmIp).toInteger() + scale.toInteger()
    flow.putInstances(serviceName, swarmIp, instances)
    build job: "service-redeploy", parameters: [[$class: "StringParameterValue", name: "serviceName", value: serviceName]]
}