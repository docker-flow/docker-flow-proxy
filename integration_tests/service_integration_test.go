package integration_test

import (
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"time"
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
	s.reconfigure()

	s.verifyReconfigure()
}

func (s ServiceIntegrationTestSuite) Test_Remove() {
	// Reconfigure
	s.reconfigure()
	s.verifyReconfigure()

	// Remove
	removeUrl := fmt.Sprintf(
		"http://%s:9080/v1/docker-flow-proxy/remove?serviceName=%s",
		os.Getenv("DOCKER_IP"),
		s.serviceName,
	)
	status, _ := s.sendRequest(removeUrl)
	s.Equal(200, status)
	url := fmt.Sprintf("http://%s:9000/v1/test", os.Getenv("DOCKER_IP"))
	time.Sleep(time.Millisecond * 100)
	resp, err := http.Get(url)
	s.NoError(err)
	b, _ := ioutil.ReadAll(resp.Body)
	s.NotEqual(200, resp.StatusCode, string(b))
}

// Util

func (s ServiceIntegrationTestSuite) reconfigure() {
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
	status, body := s.sendRequest(address)
	s.Equal(200, status, body)
}

func (s ServiceIntegrationTestSuite) sendRequest(address string) (status int, body string) {
	log.Printf(">> Sending request to %s", address)
	resp, err := http.Get(address)
	s.NoError(err)
	b, _ := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// Suite

func TestServiceIntegrationTestSuite(t *testing.T) {
	s := new(ServiceIntegrationTestSuite)
	if len(os.Getenv("DOCKER_IP")) == 0 {
		os.Setenv("DOCKER_IP", "localhost")
	}
	suite.Run(t, s)
}
