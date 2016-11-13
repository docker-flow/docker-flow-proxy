# TODO

## Certs

* Remove cert API (file and name)
* Load existing certs from labels on init
* Fix TODOs
* Document

  * API: PUT /v1/docker-flow-proxy/cert


```bash
curl -i -XPUT \
    -d 'Content of my certificate PEM file' \
    $(docker-machine ip docker-flow-proxy-tests):8080/v1/docker-flow-proxy/cert?certName=viktor.pem
```

## Content

* https://www.youtube.com/watch?v=oP0_H_UkkGA