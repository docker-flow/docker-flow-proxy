# TODO

## Certs

* Get cert files and names from other instances on init
* Remove cert API (file and name)
* Fix TODOs
* Load existing certs from labels on init
* Document

  * API: PUT /v1/docker-flow-proxy/cert

* Test

```bash
curl -i -XPUT \
    -d 'Content of my certificate PEM file' \
    $(docker-machine ip docker-flow-proxy-tests):8080/v1/docker-flow-proxy/cert?certName=viktor.pem
```

## Content

* https://www.youtube.com/watch?v=oP0_H_UkkGA