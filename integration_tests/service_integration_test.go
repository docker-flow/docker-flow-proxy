package integration_test

import (
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
)

type ServiceIntegrationTestSuite struct {
	suite.Suite
	hostIp      string
	serviceName string
}

func (s *ServiceIntegrationTestSuite) SetupTest() {
	s.serviceName = "app-1"
}

// Integration

func (s ServiceIntegrationTestSuite) Test_Reconfigure() {
	s.reconfigure("", "", "", "/v1/test")

	s.verifyReconfigure()
}

// Util

func (s ServiceIntegrationTestSuite) reconfigure(pathType, consulTemplateFePath, consulTemplateBePath string, paths ...string) {
	var address string
	address = fmt.Sprintf(
		"http://%s:9080/v1/docker-flow-proxy/reconfigure?serviceName=%s&servicePath=/v1/test&port=8080",
		os.Getenv("DOCKER_IP"),
		s.serviceName,
	)

	s.validateRequest(address)
}

func (s ServiceIntegrationTestSuite) verifyReconfigure() {
	address := fmt.Sprintf("http://%s:9000/v1/test", os.Getenv("DOCKER_IP"))

	s.validateRequest(address)
}

func (s ServiceIntegrationTestSuite) validateRequest(address string) {
	log.Printf(">> Sending request to %s", address)
	resp, err := http.Get(address)
	s.NoError(err)
	body, _ := ioutil.ReadAll(resp.Body)
	s.Equal(200, resp.StatusCode, string(body))
}

// Suite

func TestServiceIntegrationTestSuite(t *testing.T) {
	s := new(ServiceIntegrationTestSuite)
	if len(os.Getenv("DOCKER_IP")) == 0 {
		os.Setenv("DOCKER_IP", "localhost")
	}
	suite.Run(t, s)
}
