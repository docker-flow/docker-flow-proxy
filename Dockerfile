FROM haproxy:1.6-alpine
MAINTAINER 	Viktor Farcic <viktor@farcic.com>

RUN apk add --no-cache --virtual .build-deps curl unzip && \
    curl -SL https://releases.hashicorp.com/consul-template/0.13.0/consul-template_0.13.0_linux_amd64.zip -o /usr/local/bin/consul-template.zip && \
    unzip /usr/local/bin/consul-template.zip -d /usr/local/bin/ && \
    rm -f /usr/local/bin/consul-template.zip && \
    chmod +x /usr/local/bin/consul-template && \
    apk del .build-deps

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN mkdir -p /cfg/tmpl
RUN mkdir /consul_templates

ENV CONSUL_ADDRESS ""
ENV PROXY_INSTANCE_NAME "docker-flow"
ENV MODE "default"
ENV SERVICE_NAME "proxy"
ENV LISTENER_ADDRESS ""

EXPOSE 80
EXPOSE 443
EXPOSE 8080

CMD ["docker-flow-proxy", "server"]

COPY haproxy.cfg /cfg/haproxy.cfg
COPY haproxy.tmpl /cfg/tmpl/haproxy.tmpl
COPY docker-flow-proxy /usr/local/bin/docker-flow-proxy
RUN chmod +x /usr/local/bin/docker-flow-proxy
