How to build
============

docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.6 bash -c "cd /usr/src/myapp && go get -d -v -t && go test --cover -v ./... --run UnitTest && go build -v -o docker-flow-proxy"