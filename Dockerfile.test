FROM thomasjpfan/docker-flow-proxy-test-base

COPY . /src
WORKDIR /src
RUN chmod +x /src/run-tests.sh
RUN go get -d -v

CMD ["sh", "-c", "/src/run-tests.sh"]