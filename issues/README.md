```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml test

# DFP > jboss

open "http://localhost"

# DFP > tomcat

curl -i "http://localhost/tomcat"

# tomcat

open "http://localhost:8080"
```