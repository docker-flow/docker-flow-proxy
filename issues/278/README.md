```bash
docker network create -d overlay proxy

docker stack deploy -c proxy-stack.yml proxy

docker stack ps proxy

docker service logs proxy_proxy

docker network create -d overlay swarmnet

docker stack deploy -c nginx-stack.yml nginx

docker stack ps nginx

docker service logs -f proxy_proxy
```


{"Mode":"swarm","Status":"NOK","Message":"When MODE is set to \"service\" or \"swarm\", the port query is mandatory","ServiceName":"nginx_varnish","AclName":"","AddReqHeader":null,"AddResHeader":null,"BackendExtra":"","CheckResolvers":false,"ConnectionMode":"","ConsulTemplateBePath":"","ConsulTemplateFePath":"","Debug":false,"DebugFormat":"","DelReqHeader":null,"DelResHeader":null,"Distribute":true,"HttpsOnly":false,"HttpsPort":0,"IsDefaultBackend":false,"OutboundHostname":"","PathType":"","ProxyMode":"","RedirectWhenHttpProto":false,"ReqPathReplace":"","ReqPathSearch":"","ServiceCert":"","ServiceDomainMatchAll":false,"SetReqHeader":null,"SetResHeader":null,"SslVerifyNone":false,"TemplateBePath":"","TemplateFePath":"","TimeoutServer":"","TimeoutTunnel":"","UseGlobalUsers":false,"Users":null,"XForwardedProto":false,"ServiceColor":"","ServicePort":"","AclCondition":"","DomainFunction":"","FullServiceName":"","Host":"","LookupRetry":0,"LookupRetryInterval":0,"ServiceDest":[
    {
    "IgnoreAuthorization":false,
    "Port":"",
    "ReqMode":"http",
    "ReqModeFormatted":"",
    "ServiceDomain":["default.test.com"],
    "ServicePath":["/"],
    "SrcPort":0,
    "SrcPortAcl":"",
    "SrcPortAclName":"",
    "VerifyClientSsl":false,
    "UserAgent":{"Value":null,"AclName":""}
    },{
    "IgnoreAuthorization":false,
    "Port":"80","ReqMode":"http","ReqModeFormatted":"","ServiceDomain":[],"ServicePath":["/"],"SrcPort":80,"SrcPortAcl":"","SrcPortAclName":"","VerifyClientSsl":false,"UserAgent":{"Value":null,"AclName":""}},{"IgnoreAuthorization":false,"Port":"443","ReqMode":"http","ReqModeFormatted":"","ServiceDomain":[],"ServicePath":["/"],"SrcPort":443,"SrcPortAcl":"","SrcPortAclName":"","VerifyClientSsl":false,"UserAgent":{"Value":null,"AclName":""}}
]}/