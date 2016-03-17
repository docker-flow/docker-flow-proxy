FROM haproxy:1.6

RUN apt-get update && \
    apt-get install -y wget unzip && \
    wget https://releases.hashicorp.com/consul-template/0.13.0/consul-template_0.13.0_linux_amd64.zip -O /usr/local/bin/consul-template.zip && \
    unzip /usr/local/bin/consul-template.zip -d /usr/local/bin/ && \
    chmod +x /usr/local/bin/consul-template && \
    apt-get purge -y wget unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /cfg/tmpl
COPY haproxy.cfg /cfg/haproxy.cfg
COPY haproxy.tmpl /cfg/tmpl/haproxy.tmpl
COPY docker-flow-proxy /usr/local/bin/docker-flow-proxy
RUN chmod +x /usr/local/bin/docker-flow-proxy

ENV CONSUL_ADDRESS ""
EXPOSE 80
EXPOSE 8080

CMD ["docker-flow-proxy", "server"]
