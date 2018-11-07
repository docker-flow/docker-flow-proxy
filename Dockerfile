FROM golang:1.11.0-alpine AS build
ADD . /src
WORKDIR /src
RUN set -x \
    && apk add --update --no-cache --no-progress git g++ \
    && go get -d -v \
    && go test --cover ./... --run UnitTest \
    && go build -v -o docker-flow-proxy


FROM haproxy:1.8.13-alpine
LABEL org.opencontainers.image.title="Docker Flow Proxy" \
    org.opencontainers.image.description="Automated HAProxy Reverse Proxy for Docker" \
    org.opencontainers.image.url="https://proxy.dockerflow.com" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.authors="Viktor Farcic <viktor@farcic.com>" \
    org.opencontainers.image.source="https://github.com/docker-flow/docker-flow-proxy"

RUN apk --update --no-cache --no-progress add tini \
    && mkdir -p /cfg/tmpl /templates /certs /logs

ENV CERTS="" \
    CAPTURE_REQUEST_HEADER="" \
    CFG_TEMPLATE_PATH="/cfg/tmpl/haproxy.tmpl" \
    CHECK_RESOLVERS=false RESOLVERS="nameserver dns 127.0.0.11:53" \
    CONNECTION_MODE="http-keep-alive" \
    CRT_LIST_PATH="" \
    DEBUG="false" \
    DEFAULT_PORTS="80,443:ssl" \
    DEFAULT_REQ_MODE="http" \
    DO_NOT_RESOLVE_ADDR="false" \
    ENABLE_H2="true" \
    FILTER_PROXY_INSTANCE_NAME="false" \
    HEALTHCHECK="true" \
    HTTPS_ONLY="false" \
    EXTRA_FRONTEND="" \
    LISTENER_ADDRESS="" \
    MODE="default" \
    PROXY_INSTANCE_NAME="docker-flow" \
    RELOAD_ATTEMPTS="5" \
    RELOAD_INTERVAL="5000" REPEAT_RELOAD=false \
    RECONFIGURE_ATTEMPTS="20" \
    SEPARATOR="," \
    SERVICE_NAME="proxy" SERVICE_DOMAIN_ALGO="hdr_beg(host)" \
    STATS_USER="" STATS_USER_ENV="STATS_USER" STATS_PASS="" STATS_PASS_ENV="STATS_PASS" STATS_URI="" STATS_URI_ENV="STATS_URI" STATS_PORT="" \
    TIMEOUT_HTTP_REQUEST="5" TIMEOUT_HTTP_KEEP_ALIVE="15" TIMEOUT_CLIENT="20" TIMEOUT_CONNECT="5" TIMEOUT_QUEUE="30" TIMEOUT_SERVER="20" TIMEOUT_TUNNEL="3600" \
    USERS="" \
    SKIP_ADDRESS_VALIDATION="true" \
    SSL_BIND_OPTIONS="no-sslv3" \
    SSL_BIND_CIPHERS="ECDH+AESGCM:ECDH+CHACHA20:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:RSA+AESGCM:RSA+AES:!aNULL:!MD5:!DSS"

EXPOSE 80 \
    443 \
    8080

ENTRYPOINT ["/sbin/tini", "-g", "--"]
CMD ["docker-flow-proxy", "server"]
HEALTHCHECK --interval=5s --start-period=3s --timeout=10s CMD check.sh

COPY scripts/check.sh /usr/local/bin/check.sh
COPY errorfiles /errorfiles
COPY haproxy.cfg /cfg/haproxy.cfg
COPY haproxy.tmpl /cfg/tmpl/haproxy.tmpl
COPY --from=build /src/docker-flow-proxy /usr/local/bin/docker-flow-proxy
RUN chmod +x /usr/local/bin/docker-flow-proxy /usr/local/bin/check.sh
