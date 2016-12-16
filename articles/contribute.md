# Contributing To The Project

I encourage you to contribute to the *Docker Flow: Proxy*.

The project is developed using *Test Driven Development* and *Continuous Deployment* process. Test tests are divided into unit and integration tests. Every code file has an equivalent with tests (e.g. `reconfigure.go` and `reconfigure_test.go`). Ideally, I expect you to write a test that defines that should be developed, run all the unit tests and confirm that the test fails, write just enough code to make the test pass, repeat. If you are new to testing, feel free to create a pull request indicating that tests are missing and I'll help you out.

Once you are finish implementing a new feature or fixing a bug, run the *Complete Cycle*. You'll find the instructions below.

## Unit Testing

```bash
go test ./... -cover -run UnitTest
```

## Building

```bash
docker build -t vfarcic/docker-flow-proxy .
```

## Complete Cycle (Unit, Build, Staging)

### Setup

```bash
docker-machine create -d virtualbox tests

eval $(docker-machine env tests)

export HOST_IP=$(docker-machine ip tests)
```

### Unit Tests & Build

```bash
docker-compose -f docker-compose-test.yml run --rm unit
```

### Staging (Integration) Tests

```bash
docker-compose -f docker-compose-test.yml up -d staging-dep

docker-compose -f docker-compose-test.yml run --rm staging
```

### Cleanup

```bash
docker-machine rm -f tests
```