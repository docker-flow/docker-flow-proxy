```bash
docker stack deploy -c stack.yml web

docker stack ps web # Wait until all the services are running

docker service logs web_proxy

curl "http://localhost/demo/hello"
```



web_proxy.1.m2ggjp8rabp1@moby    | frontend services
web_proxy.1.m2ggjp8rabp1@moby    |     bind *:80
web_proxy.1.m2ggjp8rabp1@moby    |     bind *:443
web_proxy.1.m2ggjp8rabp1@moby    |     mode http
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |     acl url_web_rest8080_0 path_beg /demo
web_proxy.1.m2ggjp8rabp1@moby    |     use_backend web_rest-be8080_0 if url_web_rest8080_0
web_proxy.1.m2ggjp8rabp1@moby    |     acl url_web_webui80_0 path_beg /
web_proxy.1.m2ggjp8rabp1@moby    |     acl is_web_webui_http hdr(X-Forwarded-Proto) http
web_proxy.1.m2ggjp8rabp1@moby    |     redirect scheme https if is_web_webui_http url_web_webui80
web_proxy.1.m2ggjp8rabp1@moby    |     use_backend web_webui-be80_0 if url_web_webui80_0
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    | backend web_rest-be8080_0
web_proxy.1.m2ggjp8rabp1@moby    |     mode http
web_proxy.1.m2ggjp8rabp1@moby    |     server web_rest web_rest:8080
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    |
web_proxy.1.m2ggjp8rabp1@moby    | backend web_webui-be80_0
web_proxy.1.m2ggjp8rabp1@moby    |     mode http
web_proxy.1.m2ggjp8rabp1@moby    |     redirect scheme https if !{ ssl_fc }
web_proxy.1.m2ggjp8rabp1@moby    |     server web_webui web_webui:80