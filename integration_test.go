// +build integration

package main

// $ export HOST_IP=<HOST_IP>

// Unit tests
// $ docker-compose -f docker-compose-test.yml run --rm unit

// Build
// $ docker-compose -f docker-compose-test.yml down
// $ docker-compose -f docker-compose.yml build app

// Staging tests
// $ docker-compose -f docker-compose-test.yml up -d staging-dep
// $ docker-compose -f docker-compose-test.yml run --rm staging
// $ docker-compose -f docker-compose-test.yml down

// Push
// $ docker push vfarcic/docker-flow-proxy

// Production tests
// $ docker-compose -f docker-compose-test.yml up -d staging-dep
// $ docker-compose -f docker-compose-test.yml run --rm production

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"io/ioutil"
)

type IntegrationTestSuite struct {
	suite.Suite
	hostIp string
	serviceName string
}

func (s *IntegrationTestSuite) SetupTest() {
	s.serviceName = "test-service"
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
	resp, err := http.Get(url)
	s.NoError(err)
	s.NotEqual(200, resp.StatusCode)
}

func (s IntegrationTestSuite) Test_PutToConsul() {
	s.reconfigure("", "/v1/test")

	url := fmt.Sprintf(
		"http://%s:8500/v1/kv/docker-flow/%s/path?raw",
		os.Getenv("DOCKER_IP"),
		s.serviceName,
	)
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	s.Equal("/v1/test", string(body))
}

// Util

func (s IntegrationTestSuite) verifyReconfigure(version int) {
	address := fmt.Sprintf("http://%s/v%d/test", os.Getenv("DOCKER_IP"), version)
	logPrintf("Sending verify request to %s", address)
	resp, err := http.Get(address)

	s.NoError(err)
	s.Equal(200, resp.StatusCode)
}

func (s IntegrationTestSuite) reconfigure(pathType string, paths ...string) {
	address := fmt.Sprintf(
		"http://%s:8080/v1/docker-flow-proxy/reconfigure?serviceName=%s&servicePath=%s&pathType=%s",
		os.Getenv("DOCKER_IP"),
		s.serviceName,
		strings.Join(paths, ","),
		pathType,
	)
	logPrintf("Sending reconfigure request to %s", address)
	_, err := http.Get(address)
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
