package integration_test

/*
Setup
$ docker-machine create -d virtualbox docker-flow-proxy-tests
$ eval $(docker-machine env docker-flow-proxy-tests)
$ export HOST_IP=$(docker-machine ip docker-flow-proxy-tests)

Unit tests
$ docker-compose -f docker-compose-test.yml run --rm unit

Build
$ docker-compose build app

Staging tests
$ docker-compose -f docker-compose-test.yml up -d staging-dep
$ docker-compose -f docker-compose-test.yml run --rm staging
$ docker-compose -f docker-compose-test.yml down

Push
$ docker push vfarcic/docker-flow-proxy

Production tests
$ docker-compose -f docker-compose-test.yml up -d staging-dep
$ docker-compose -f docker-compose-test.yml run --rm production

Manual tests
$ docker-compose -f docker-compose-test.yml up -d staging-dep
$ curl -i "localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=test-service&servicePath=/v1/test"
$ curl -i localhost/v1/test
$ curl -i "localhost:8080/v1/docker-flow-proxy/reconfigure?serviceName=test-service&servicePath=^/v1/.*es.*&pathType=path_reg"
$ docker-compose -f docker-compose-test.yml down

Cleanup
$ docker-machine rm -f docker-flow-proxy-tests
*/

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"log"
	"os/exec"
)

type IntegrationTestSuite struct {
	suite.Suite
	hostIp      string
	serviceName string
}

func (s *IntegrationTestSuite) SetupTest() {
	s.serviceName = "test-service"
}

// Integration

func (s IntegrationTestSuite) Test_Reconfigure_MultipleInstances() {
	s.reconfigure("", "", "", "/v1/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Reconfigure_MultipleServices() {
	s.reconfigure("", "", "", "/v1/test")
	s.serviceName = "test-other-service"
	s.reconfigure("", "", "", "/v2/test")

	s.verifyReconfigure(1)
	s.verifyReconfigure(2)
}

func (s IntegrationTestSuite) Test_Reconfigure_PathReg() {
	s.reconfigure("path_reg", "", "", "/.*/test")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Reconfigure_MultiplePaths() {
	s.reconfigure("", "", "/v1/test", "", "/v2/test")

	s.verifyReconfigure(2)
}

func (s IntegrationTestSuite) Test_Remove() {
	s.reconfigure("", "", "", "/v1/test")
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
	s.reconfigure("", "", "", "/v1/test")

	url := fmt.Sprintf(
		"http://%s:8500/v1/kv/proxy-test-instance/%s/path?raw",
		os.Getenv("DOCKER_IP"),
		s.serviceName,
	)
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	s.Equal("/v1/test", string(body))
}

func (s IntegrationTestSuite) Test_Reconfigure_ConsulTemplatePath() {
	http.Get(fmt.Sprintf(
		"http://%s:8080/v1/docker-flow-proxy/remove?serviceName=test-service",
		os.Getenv("DOCKER_IP"),
	))
	s.reconfigure("", "/test_configs/tmpl/my-service-fe.tmpl", "/test_configs/tmpl/my-service-be.tmpl", "")

	s.verifyReconfigure(1)
}

func (s IntegrationTestSuite) Test_Config() {
	resp, _ := http.Get(fmt.Sprintf(
		"http://%s:8080/v1/docker-flow-proxy/config",
		os.Getenv("DOCKER_IP"),
	))
	cmdString := "docker cp dockerflowproxy_staging-dep_1:/cfg/haproxy.cfg /tmp/"
	exec.Command("/bin/sh", "-c", cmdString).Output()

	expected, _ := ioutil.ReadFile("/tmp/haproxy.cfg")

	body, _ := ioutil.ReadAll(resp.Body)

	s.Equal(string(expected[:]), string(body))
}

// Util

func (s IntegrationTestSuite) verifyReconfigure(version int) {
	address := fmt.Sprintf("http://%s/v%d/test", os.Getenv("DOCKER_IP"), version)
	log.Printf(">> Sending verify request to %s", address)
	resp, err := http.Get(address)

	s.NoError(err)
	s.Equal(200, resp.StatusCode)
}

func (s IntegrationTestSuite) reconfigure(pathType, consulTemplateFePath, consulTemplateBePath string, paths ...string) {
	var address string
	if len(consulTemplateFePath) > 0 {
		address = fmt.Sprintf(
			"http://%s:8080/v1/docker-flow-proxy/reconfigure?serviceName=%s&consulTemplateFePath=%s&consulTemplateBePath=%s",
			os.Getenv("DOCKER_IP"),
			s.serviceName,
			consulTemplateFePath,
			consulTemplateBePath,
		)
	} else {
		address = fmt.Sprintf(
			"http://%s:8080/v1/docker-flow-proxy/reconfigure?serviceName=%s&servicePath=%s&pathType=%s",
			os.Getenv("DOCKER_IP"),
			s.serviceName,
			strings.Join(paths, ","),
			pathType,
		)
	}
	log.Printf(">> Sending reconfigure request to %s", address)
	_, err := http.Get(address)
	s.NoError(err)
}

// Suite

func TestGeneralIntegrationTestSuite(t *testing.T) {
	s := new(IntegrationTestSuite)
	if len(os.Getenv("DOCKER_IP")) == 0 {
		os.Setenv("DOCKER_IP", "localhost")
	}
	if len(os.Getenv("CONSUL_IP")) == 0 {
		os.Setenv("CONSUL_IP", "localhost")
	}
	suite.Run(t, s)
}
