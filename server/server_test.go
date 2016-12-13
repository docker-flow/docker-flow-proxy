package server

import (
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type ServerTestSuite struct {
	BaseUrl        string
	ReconfigureUrl string
	ServiceName    string
	Server         *httptest.Server
	DnsIps         []string
	suite.Suite
}

func (s *ServerTestSuite) SetupTest() {
}

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	s.ServiceName = "my-fancy-service"
	s.BaseUrl = "/v1/docker-flow-proxy/reconfigure"
	s.ReconfigureUrl = fmt.Sprintf(
		"%s?serviceName=%s&serviceColor=pink&servicePath=/path/to/my/service/api&serviceDomain=my-domain.com",
		s.BaseUrl,
		s.ServiceName,
	)

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/reconfigure":
				if strings.EqualFold(r.URL.Query().Get("returnError"), "true") {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
				}
			case "/v1/docker-flow-proxy/remove":
				if strings.EqualFold(r.URL.Query().Get("returnError"), "true") {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { s.Server.Close() }()

	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return s.DnsIps, nil
	}

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	addr := strings.Replace(s.Server.URL, "http://", "", -1)
	s.DnsIps = []string{strings.Split(addr, ":")[0]}

	suite.Run(t, s)
}

// SendDistributeRequests

func (s *ServerTestSuite) Test_SendDistributeRequests_InvokesLookupHost() {
	var actualHost string
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		actualHost = host
		return []string{}, nil
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{}
	srv.SendDistributeRequests(req, "8080", s.ServiceName)

	s.Assert().Equal(fmt.Sprintf("tasks.%s", s.ServiceName), actualHost)
}

func (s *ServerTestSuite) Test_SednDistributeRequests_ReturnsError_WhenLookupHostFails() {
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return []string{}, fmt.Errorf("This is an LookupHost error")
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{}
	status, err := srv.SendDistributeRequests(req, "8080", s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIp() {
	var actualPath string
	var actualQuery url.Values
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualQuery = r.URL.Query()
		actualPath = r.URL.Path
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal(s.BaseUrl, actualPath)
	s.Assert().Equal("false", actualQuery.Get("distribute"))
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIpWithTheCorrectMethod() {
	actualProtocol := "GET"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualProtocol = r.Method
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, nil)

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal("PUT", actualProtocol)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIpWithTheBody() {
	actualBody := ""
	expectedBody := "THIS IS BODY"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		actualBody = string(body)
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, strings.NewReader(expectedBody))

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal(expectedBody, actualBody)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_ReturnsError_WhenRequestFail() {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer func() { testServer.Close() }()

	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	status, err := srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

// Mocks

type ServerMock struct {
	mock.Mock
}

func (m *ServerMock) SendDistributeRequests(req *http.Request, port, serviceName string) (status int, err error) {
	params := m.Called(req, port, serviceName)
	return params.Int(0), params.Error(1)
}

func getServerMock(skipMethod string) *ServerMock {
	mockObj := new(ServerMock)
	if skipMethod != "SendDistributeRequests" {
		mockObj.On("SendDistributeRequests", mock.Anything, mock.Anything, mock.Anything).Return(200, nil)
	}
	return mockObj
}
