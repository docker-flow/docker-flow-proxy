```bash
docker network create -d overlay proxy

docker stack deploy -c stack.yml proxy

curl -i -H "Host: acme.com" "http://localhost/demo/hello"
```

```
HTTP/1.1 301 Moved Permanently
Content-length: 0
Location: http://google.com/demo/hello
```
