```
docker service update --env-add EXTRA_FRONTEND="option http-buffer-request,capture request header Referrer len 64" proxy_proxy
```
