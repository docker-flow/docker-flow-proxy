// +build integration

package main

// Without Docker Machine
// $ export DOCKER_IP=<HOST_IP>
// $ export CONSUL_IP=<HOST_IP>

// Unit tests
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml run --rm tests-unit

// Build
// $ docker-compose -f docker-compose.yml build proxy

// Staging tests
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml down
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml up -d tests-staging-dep
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml run --rm tests-staging
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml down

// Push
// $ docker push vfarcic/docker-flow-proxy

// Production tests
// $ docker-compose -p docker-flow-proxy-test -f docker-compose-test.yml run --rm tests-production

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type IntegrationTestSuite struct {
	suite.Suite
}

func (s *IntegrationTestSuite) SetupTest() {
}

// Integration

func (s IntegrationTestSuite) Test_Reconfigure_MultipleInstances() {
	s.reconfigure("", "/v1/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Reconfigure_PathReg() {
	s.reconfigure("path_reg", "/.*/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Reconfigure_MultiplePaths() {
	s.reconfigure("", "/v1/test", "/v2/test")

	s.verifyReconfigure(2)
}

func (s IntegrationTestSuite) Test_Remove() {
	s.reconfigure("", "/v1/test")
	s.verifyReconfigure(1)

	_, err := http.Get(fmt.Sprintf(
		"http://%s:8080/v1/docker-flow-proxy/remove?serviceName=test-service",
		os.Getenv("DOCKER_IP"),
	))

	s.NoError(err)
	url := fmt.Sprintf("http://%s/v1/test", os.Getenv("DOCKER_IP"))
	fmt.Println(url)
	resp, err := http.Get(url)
	s.NoError(err)
	s.NotEqual(200, resp.StatusCode)
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
	suite.Run(t, s)
}
