// +build integration
package main

// To run locally on OS X
// $ docker-machine create -d virtualbox testing
// $ eval $(docker-machine env testing)
// $ export DOCKER_IP=$(docker-machine ip testing)
// $ export CONSUL_IP=$(docker-machine ip testing)
// $ docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v $GOPATH:/go golang:1.6 go build -v -o docker-flow-proxy
// $ docker build -t vfarcic/docker-flow-proxy .
// $ go build && go test --cover --tags integration
// $ docker-machine rm -f testing

// TODO: Change books-ms for a lighter service

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
	"os/exec"
	"net/http"
	"strings"
	"os"
)

type IntegrationTestSuite struct {
	suite.Suite
}

func (s *IntegrationTestSuite) SetupTest() {
}

// Integration

func (s IntegrationTestSuite) Test_Integration_SingleInstance() {
	s.reconfigure("","/v1/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Integration_MultipleInstances() {
	if ok := s.runCmd(
		"docker-compose",
		"-p", "test-service",
		"-f", "docker-compose-test.yml",
		"scale", "app=3",
	); !ok {
		s.Fail("Failed to scale the service")
	}

	s.reconfigure("", "/v1/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Integration_PathReg() {
	if ok := s.runCmd(
		"docker-compose",
		"-p", "test-service",
		"-f", "docker-compose-test.yml",
		"scale", "app=3",
	); !ok {
		s.Fail("Failed to scale the service")
	}

	s.reconfigure("path_reg", "/.*/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Integration_MultiplePaths() {
	s.reconfigure("", "/v1/test", "/v2/test")

	s.verifyReconfigure(2)
}

// Util

func (s IntegrationTestSuite) verifyReconfigure(version int) {
	url := fmt.Sprintf("http://%s/v%d/test", os.Getenv("DOCKER_IP"), version)
	resp, err := http.Get(url)

	s.NoError(err)
	s.Equal(200, resp.StatusCode)
}

func (s IntegrationTestSuite) reconfigure(pathType string, paths ...string) {
	_, err := http.Get(fmt.Sprintf(
		"http://%s:8080/v1/docker-flow-proxy/reconfigure?serviceName=test-service&servicePath=%s&pathType=%s",
		os.Getenv("DOCKER_IP"),
		strings.Join(paths, ","),
		pathType,
	))
	s.NoError(err)
}

func (s IntegrationTestSuite) runCmd(command string, args ...string) bool {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("%s %s\n%s\n", command, strings.Join(args, " "), err.Error())
		return false
	}
	return true
}

// Suite

func TestIntegrationTestSuite(t *testing.T) {
	s := new(IntegrationTestSuite)
	if len(os.Getenv("DOCKER_IP")) == 0 {
		os.Setenv("DOCKER_IP", "localhost")
	}
	if len(os.Getenv("CONSUL_IP")) == 0 {
		os.Setenv("CONSUL_IP", "localhost")
	}
	ok := s.runCmd("docker-compose", "up", "-d", "consul", "proxy", "registrator")
	if !ok {
		s.FailNow("Could not run consul, proxy, and registrator")
	}
	ok = s.runCmd("docker-compose", "-p", "test-service", "-f", "docker-compose-test.yml", "up", "-d")
	if !ok {
		s.FailNow("Could not run the test service")
	}
	suite.Run(t, s)
}