package registry

import (
	"testing"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"net/http"
	"io/ioutil"
	"fmt"
	"strings"
	"sync"
)

type ConsulTestSuite struct {
	suite.Suite
	registry Registry
}

// PutService

func (s *ConsulTestSuite) Test_PutService_PutsDataToConsul() {
	instanceName := "my-instance"
	var actualUrl, actualBody, actualMethod []string
	var mu = &sync.Mutex{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		mu.Lock()
		actualMethod = append(actualMethod, r.Method)
		actualUrl = append(actualUrl, r.URL.Path)
		actualBody = append(actualBody, string(body))
		mu.Unlock()
	}))
	defer server.Close()
	err := Consul{}.PutService(server.URL, instanceName, s.registry)

	s.NoError(err)

	type data struct{ key, value string }

	d := []data{
		data{"color", s.registry.ServiceColor},
		data{"path", strings.Join(s.registry.ServicePath, ",")},
		data{"domain", s.registry.ServiceDomain},
		data{"pathtype", s.registry.PathType},
		data{"skipcheck", fmt.Sprintf("%t", s.registry.SkipCheck)},
		data{"consultemplatefepath", s.registry.ConsulTemplateFePath},
		data{"consultemplatebepath", s.registry.ConsulTemplateBePath},
	}
	for _, e := range d {
		s.Contains(actualUrl, fmt.Sprintf("/v1/kv/%s/%s/%s", instanceName, s.registry.ServiceName, e.key))
		s.Contains(actualBody, e.value)
		s.Equal("PUT", actualMethod[0])
	}
}

func (s *ConsulTestSuite) Test_PutService_ReturnsError_WhenFailure() {
	err := Consul{}.PutService("http:///THIS/URL/DOES/NOT/EXIST", "my-instance", s.registry)

	s.Error(err)
}

func (s *ConsulTestSuite) Test_SendPutRequest_AddsHttp_WhenNotPresent() {
	instanceName := "my-proxy-instance"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer server.Close()

	err := Consul{}.PutService(strings.Replace(server.URL, "http://", "", -1), instanceName, s.registry)

	s.NoError(err)
}

// SendPutRequest

func (s *ConsulTestSuite) Test_SendPutRequest_SendsDataToConsul() {
	instanceName := "my-proxy-instance"
	key := "my-key"
	value := "my-value"
	serviceName := "my-service"
	var actualUrl, actualBody, actualMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		actualMethod = r.Method
		actualUrl = r.URL.Path
		actualBody = string(body)
	}))
	defer server.Close()

	c := make(chan error)
	go Consul{}.SendPutRequest(server.URL, serviceName, key, value, instanceName, c)
	err := <-c

	s.NoError(err)
	s.Equal(fmt.Sprintf("/v1/kv/%s/%s/%s", instanceName, serviceName, key), actualUrl)
	s.Equal(value, actualBody)
	s.Equal("PUT", actualMethod)
}

// DeleteService

func (s *ConsulTestSuite) Test_DeleteService_DeletesServiceFromConsul() {
	instanceName := "my-proxy-instance"
	var actualUrl, actualMethod, actualQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualMethod = r.Method
		actualUrl = r.URL.Path
		actualQuery = r.URL.RawQuery
	}))
	defer server.Close()

	err := Consul{}.DeleteService(server.URL, s.registry.ServiceName, instanceName)

	s.NoError(err)
	s.Equal(fmt.Sprintf("/v1/kv/%s/%s", instanceName, s.registry.ServiceName), actualUrl)
	s.Equal("DELETE", actualMethod)
	s.Equal("recurse", actualQuery)
}

func (s *ConsulTestSuite) Test_DeleteService_ReturnsError_WhenFailure() {
	err := Consul{}.DeleteService("http:///THIS/URL/DOES/NOT/EXIST", s.registry.ServiceName, "my-instance")

	s.Error(err)
}

func (s *ConsulTestSuite) Test_DeleteService_AddsHttp_WhenNotPresent() {
	instanceName := "my-proxy-instance"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer server.Close()

	err := Consul{}.DeleteService(strings.Replace(server.URL, "http://", "", -1), s.registry.ServiceName, instanceName)

	s.NoError(err)
}

// SendDeleteRequest

func (s *ConsulTestSuite) Test_SendDeleteRequest_DeletesDataFromConsul() {
	instanceName := "my-proxy-instance"
	key := "my-key"
	value := "my-value"
	serviceName := "my-service"
	var actualUrl, actualBody, actualMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		actualMethod = r.Method
		actualUrl = r.URL.Path
		actualBody = string(body)
	}))
	defer server.Close()

	c := make(chan error)
	go Consul{}.SendDeleteRequest(server.URL, serviceName, key, value, instanceName, c)
	err := <-c

	s.NoError(err)
	s.Equal(fmt.Sprintf("/v1/kv/%s/%s/%s", instanceName, serviceName, key), actualUrl)
	s.Equal(value, actualBody)
	s.Equal("DELETE", actualMethod)
}

// Suite

func TestTestTestSuite(t *testing.T) {
	s := new(ConsulTestSuite)
	s.registry = Registry{
		ServiceName: "my-service",
		ServiceColor: "ServiceColor",
		ServicePath: []string{"pat1", "path2"},
		ServiceDomain: "ServiceDomain",
		PathType: "PathType",
		SkipCheck: true,
		ConsulTemplateFePath: "ConsulTemplateFePath",
		ConsulTemplateBePath: "ConsulTemplateBePath",
	}
	suite.Run(t, s)
}