package server

import (
	"../proxy"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"encoding/json"
	"../actions"
)

type ServerTestSuite struct {
	BaseUrl        string
	ReconfigureUrl string // TODO: Remove
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

	SendDistributeRequests(req, "8080", s.ServiceName)

	s.Assert().Equal(fmt.Sprintf("tasks.%s", s.ServiceName), actualHost)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_ReturnsError_WhenLookupHostFails() {
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return []string{}, fmt.Errorf("This is an LookupHost error")
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	status, err := SendDistributeRequests(req, "8080", s.ServiceName)

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

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	SendDistributeRequests(req, port, s.ServiceName)

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

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, nil)

	SendDistributeRequests(req, port, s.ServiceName)

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

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, strings.NewReader(expectedBody))

	SendDistributeRequests(req, port, s.ServiceName)

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

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	status, err := SendDistributeRequests(req, port, s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

// TestHandler

func (s *ServerTestSuite) Test_TestHandler_ReturnsStatus200() {
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/v%d/test", ver), nil)

		srv := Serve{}
		srv.TestHandler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

// ReconfigureHandler

func (s *ServerTestSuite) Test_ReconfigureHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reconfigure?serviceName=my-service", nil)

	srv := Serve{}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesErrorHeader_WhenReconfigureDistributeIsTrueAndError() {
	serve := Serve{}
	serve.Port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := SendDistributeRequests
	defer func() { SendDistributeRequests = sendDistributeRequestsOrig }()
	SendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, fmt.Errorf("This is an error")
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesStatusOK_WhenReconfigureDistributeIsTrue() {
	serve := Serve{}
	serve.Port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := SendDistributeRequests
	defer func() { SendDistributeRequests = sendDistributeRequestsOrig }()
	SendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, nil
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenServiceNameQueryIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus200_WhenReqModeIsTcp() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock {
			ExecuteMock: func(args []string) error {
				return nil
			},
		}
	}

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndSrcPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenServicePathQueryIsNotPresent() {
	url := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service"
	req, _ := http.NewRequest("GET", url, nil)
	rw := getResponseWriterMock()

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenModeIsServiceAndPortIsNotPresent() {
	url := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&serviceColor=orange&servicePath=/demo&serviceDomain=my-domain.com"
	req, _ := http.NewRequest("GET", url, nil)
	rw := getResponseWriterMock()

	srv := Serve{Mode: "service"}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenModeIsSwarmAndPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&serviceColor=orange&servicePath=/demo&serviceDomain=my-domain.com"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := Serve{Mode: "swARM"}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesReconfigureExecute() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath.1=/demo&port.1=1234"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock {
			ExecuteMock: func(args []string) error {
				invoked = true
				return nil
			},
		}
	}

	srv := Serve{Mode: "swarm"}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_DoesNotInvokeReconfigureExecute_WhenDistributeIsTrue() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock {
			ExecuteMock: func(args []string) error {
				invoked = true
				return nil
			},
		}
	}

	srv := Serve{Mode: "swarm"}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.False(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus500_WhenReconfigureExecuteFails() {
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock {
			ExecuteMock: func(args []string) error {
				return fmt.Errorf("This is an error")
			},
		}
	}
	rw := getResponseWriterMock()
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234"
	req, _ := http.NewRequest("GET", addr, nil)

	srv := Serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCert_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := Serve{
		Mode: "swarm",
		Cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-service", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCertWithDomainName_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceDomain=my-domain.com&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := Serve{
		Mode: "swarm",
		Cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-domain.com", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesReconfigureExecute_WhenConsulTemplatePathIsPresent() {
	sd := proxy.ServiceDest{
		ServicePath: []string{},
	}
	pathFe := "/path/to/consul/fe/template"
	pathBe := "/path/to/consul/be/template"
	var actualBase actions.BaseReconfigure
	expectedBase := actions.BaseReconfigure{
		ConsulAddresses: []string{"http://my-consul.com"},
	}
	var actualService proxy.Service
	expectedService := proxy.Service{
		ServiceName:          "my-service",
		ConsulTemplateFePath: pathFe,
		ConsulTemplateBePath: pathBe,
		ReqMode:              "http",
		ServiceDest:          []proxy.ServiceDest{sd},
	}
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	invoked := false
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		actualBase = baseData
		actualService = serviceData
		return ReconfigureMock {
			ExecuteMock: func(args []string) error {
				invoked = true
				return nil
			},
		}
	}
	serverImpl := Serve{
		ConsulAddresses: []string{"http://my-consul.com"},
	}
	addr := fmt.Sprintf(
		"/v1/docker-flow-proxy/reconfigure?serviceName=my-service&consulTemplateFePath=%s&consulTemplateBePath=%s",
		pathFe,
		pathBe,
	)
	req, _ := http.NewRequest("GET", addr, nil)

	serverImpl.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal(expectedBase, actualBase)
	s.Equal(expectedService, actualService)
	s.True(invoked)
	s.Fail("xxx")
}

// ReloadHandler

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsStatus200() {
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			return nil
		},
	}
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

		srv := Serve{}
		srv.ReloadHandler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReload() {
	invoked := false
	reloadOrig := reload
	defer func() { reload = reloadOrig }()
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			invoked = true
			return nil
		},
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

	srv := Serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithRecreateParam() {
	actualRecreate := false
	actualListenerAddr := ""
	reloadOrig := reload
	defer func() { reload = reloadOrig }()
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			actualRecreate = recreate
			actualListenerAddr = listenerAddr
			return nil
		},
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?recreate=true", nil)

	srv := Serve{}
	srv.ListenerAddress = "my-listener"
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(actualRecreate)
	s.Empty(actualListenerAddr)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithFromListenerParam() {
	actualListenerAddr := ""
	reloadOrig := reload
	defer func() { reload = reloadOrig }()
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			actualListenerAddr = listenerAddr
			return nil
		},
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?fromListener=true", nil)

	srv := Serve{}
	srv.ListenerAddress = "my-listener"
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal(srv.ListenerAddress, actualListenerAddr)
}

func (s *ServerTestSuite) Test_ReloadHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			return nil
		},
	}

	srv := Serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status: "OK",
	})
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	reload = ReloadMock{
		ExecuteMock: func(recreate bool, listenerAddr string) error {
			return nil
		},
	}
	respWriterMock := getResponseWriterMock()

	srv := Serve{}
	srv.ReloadHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

// Remove Handler

func (s *ServerTestSuite) Test_RemoveHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)

	srv := Serve{}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: "my-service",
	})
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	srv := Serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsStatus400_WhenServiceNameIsNotPresent() {
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)
	respWriterMock := getResponseWriterMock()

	srv := Serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_RemoveHandler_InvokesRemoveExecute() {
	mockObj := getRemoveMock("")
	aclName := "my-acl"
	var actual actions.Remove
	expected := actions.Remove{
		ServiceName:     s.ServiceName,
		TemplatesPath:   "",
		ConfigsPath:     "",
		ConsulAddresses: []string{"http://1.2.3.4:1234"},
		InstanceName:    "proxy-test-instance",
		AclName:         aclName,
	}
	actions.NewRemove = func(
		serviceName, aclName, configsPath, templatesPath string,
		consulAddresses []string,
		instanceName, mode string,
	) actions.Removable {
		actual = actions.Remove{
			ServiceName:     serviceName,
			AclName:         aclName,
			TemplatesPath:   templatesPath,
			ConfigsPath:     configsPath,
			ConsulAddresses: consulAddresses,
			InstanceName:    instanceName,
			Mode:            mode,
		}
		return mockObj
	}
	url := fmt.Sprintf("/v1/docker-flow-proxy/remove?serviceName=%s&aclName=%s", s.ServiceName, aclName)
	req, _ := http.NewRequest("GET", url, nil)

	srv := Serve{
		ConsulAddresses: expected.ConsulAddresses,
		ServiceName:     expected.InstanceName,
	}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal(expected, actual)
	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

func (s *ServerTestSuite) Test_RemoveHandler_WritesErrorHeader_WhenRemoveDistributeIsTrueAndItFails() {
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	serve := Serve{}
	serve.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 500)
}

// GetServiceFromUrl

func (s *ServerTestSuite) Test_GetServiceFromUrl_ReturnsProxyService() {
	expected := proxy.Service{
		ServiceName:           "serviceName",
		AclName:               "aclName",
		ServiceColor:          "serviceColor",
		ServiceCert:           "serviceCert",
		OutboundHostname:      "outboundHostname",
		ConsulTemplateFePath:  "consulTemplateFePath",
		ConsulTemplateBePath:  "consulTemplateBePath",
		PathType:              "pathType",
		ReqPathSearch:         "reqPathSearch",
		ReqPathReplace:        "reqPathReplace",
		TemplateFePath:        "templateFePath",
		TemplateBePath:        "templateBePath",
		TimeoutServer:         "timeoutServer",
		TimeoutTunnel:         "timeoutTunnel",
		ReqMode:               "reqMode",
		HttpsOnly:             true,
		XForwardedProto:       true,
		RedirectWhenHttpProto: true,
		HttpsPort:             1234,
		ServiceDomain:         []string{"domain1", "domain2"},
		SkipCheck:             true,
		Distribute:            true,
		SslVerifyNone:         true,
		ServiceDomainMatchAll: true,
		ServiceDest:           []proxy.ServiceDest{},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&aclName=%s&serviceColor=%s&serviceCert=%s&outboundHostname=%s&consulTemplateFePath=%s&consulTemplateBePath=%s&pathType=%s&reqPathSearch=%s&reqPathReplace=%s&templateFePath=%s&templateBePath=%s&timeoutServer=%s&timeoutTunnel=%s&reqMode=%s&httpsOnly=%t&xForwardedProto=%t&redirectWhenHttpProto=%t&httpsPort=%d&serviceDomain=%s&skipCheck=%t&distribute=%t&sslVerifyNone=%t&serviceDomainMatchAll=%t",
		s.BaseUrl,
		expected.ServiceName,
		expected.AclName,
		expected.ServiceColor,
		expected.ServiceCert,
		expected.OutboundHostname,
		expected.ConsulTemplateFePath,
		expected.ConsulTemplateBePath,
		expected.PathType,
		expected.ReqPathSearch,
		expected.ReqPathReplace,
		expected.TemplateFePath,
		expected.TemplateBePath,
		expected.TimeoutServer,
		expected.TimeoutTunnel,
		expected.ReqMode,
		expected.HttpsOnly,
		expected.XForwardedProto,
		expected.RedirectWhenHttpProto,
		expected.HttpsPort,
		strings.Join(expected.ServiceDomain, ","),
		expected.SkipCheck,
		expected.Distribute,
		expected.SslVerifyNone,
		expected.ServiceDomainMatchAll,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Equal(expected, actual)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_DefaultsReqModeToHttp() {
	req, _ := http.NewRequest("GET", s.BaseUrl, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Equal("http", actual.ReqMode)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_GetsUsersFromProxyExtractUsersFromString() {
	user1 := proxy.User{
		Username:      "user1",
		Password:      "pass1",
		PassEncrypted: false,
	}
	user2 := proxy.User{
		Username:      "user2",
		Password:      "pass2",
		PassEncrypted: true,
	}
	extractUsersFromStringOrig := extractUsersFromString
	defer func() { extractUsersFromString = extractUsersFromStringOrig }()
	actualServiceName := ""
	expectedServiceName := "my-service"
	actualUsers := ""
	expectedUsers := "user1,user2"
	actualUsersPassEncrypted := false
	expectedUsersPassEncrypted := true
	actualSkipEmptyPassword := true
	expectedSkipEmptyPassword := false
	extractUsersFromString = func(serviceName, usersString string, usersPassEncrypted, skipEmptyPassword bool) []*proxy.User {
		actualServiceName = serviceName
		actualUsers = usersString
		actualUsersPassEncrypted = usersPassEncrypted
		actualSkipEmptyPassword = skipEmptyPassword
		return []*proxy.User{&user1, &user2}
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&users=%s&usersPassEncrypted=%t",
		s.BaseUrl,
		expectedServiceName,
		expectedUsers,
		expectedUsersPassEncrypted,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Contains(actual.Users, user1)
	s.Contains(actual.Users, user2)
	s.Equal(expectedServiceName, actualServiceName)
	s.Equal(expectedUsers, actualUsers)
	s.Equal(expectedUsersPassEncrypted, actualUsersPassEncrypted)
	s.Equal(expectedSkipEmptyPassword, actualSkipEmptyPassword)
}

// mergeUsers

func (s *ServerTestSuite) Test_UsersMerge_AllCases() {
	usersBasePathOrig := usersBasePath
	defer func() { usersBasePath = usersBasePathOrig }()
	usersBasePath = "../test_configs/%s.txt"
	users := mergeUsers("someService", "user1:pass1,user2:pass2", "", false, "", false)
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})
	users = mergeUsers("someService", "user1:pass1,user2", "", false, "", false)
	//user without password will not be included
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
	})
	users = mergeUsers("someService", "user1:passWoRd,user2", "users", false, "", false)
	//user2 password will come from users file
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "users", true, "", false)
	//user2 password will come from users file, all encrypted
	s.Equal(users, []proxy.User{
		{PassEncrypted: true, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "users", false, "user1:pass1,user2:pass2", false)
	//user2 password will come from users file, but not from global one
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "", false, "user1:pass1,user2:pass2", false)
	//user2 password will come from global file
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "", false, "user1:pass1,user2:pass2", true)
	//user2 password will come from global file, globals encrypted only
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "", true, "user1:pass1,user2:pass2", true)
	//user2 password will come from global file, all encrypted
	s.Equal(users, []proxy.User{
		{PassEncrypted: true, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1,user2", "", false, "", false)
	//no users found dummy one generated
	s.Equal(len(users), 1)
	s.Equal(users[0].Username, "dummyUser")

	users = mergeUsers("someService", "", "users", false, "", false)
	//Users from file only
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "", "", false, "user1:pass1,user2:pass2", false)
	//No users when only globals present
	s.Equal(len(users), 0)

}

// Mocks

type ServerMock struct {
	mock.Mock
}

func (m *ServerMock) GetServiceFromUrl(sd []proxy.ServiceDest, req *http.Request) proxy.Service {
	params := m.Called(sd, req)
	return params.Get(0).(proxy.Service)
}

func (m *ServerMock) TestHandler(w http.ResponseWriter, req *http.Request) {
	m.Called(w, req)
}

func (m *ServerMock) ReloadHandler(w http.ResponseWriter, req *http.Request) {
	m.Called(w, req)
}

func (m *ServerMock) RemoveHandler(w http.ResponseWriter, req *http.Request) {
	m.Called(w, req)
}

func getServerMock(skipMethod string) *ServerMock {
	mockObj := new(ServerMock)
	if skipMethod != "ReloadHandler" {
		mockObj.On("ReloadHandler", mock.Anything, mock.Anything)
	}
	if skipMethod != "RemoveHandler" {
		mockObj.On("RemoveHandler", mock.Anything, mock.Anything)
	}
	if skipMethod != "TestHandler" {
		mockObj.On("TestHandler", mock.Anything, mock.Anything)
	}
	return mockObj
}

type ReloadMock struct {
	ExecuteMock func(recreate bool, listenerAddr string) error
}

func (m ReloadMock) Execute(recreate bool, listenerAddr string) error {
	return m.ExecuteMock(recreate, listenerAddr)
}

type RemoveMock struct {
	mock.Mock
}

func (m *RemoveMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func getRemoveMock(skipMethod string) *RemoveMock {
	mockObj := new(RemoveMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	return mockObj
}

type ReconfigureMock struct {
	ExecuteMock           func(args []string) error
	GetDataMock           func() (actions.BaseReconfigure, proxy.Service)
	ReloadAllServicesMock func(addresses []string, instanceName, mode, listenerAddress string) error
	GetTemplatesMock      func(sr *proxy.Service) (front, back string, err error)
}

func (m ReconfigureMock) Execute(args []string) error {
	return m.ExecuteMock(args)
}

func (m ReconfigureMock) GetData() (actions.BaseReconfigure, proxy.Service) {
	return m.GetDataMock()
}

func (m ReconfigureMock) ReloadAllServices(addresses []string, instanceName, mode, listenerAddress string) error {
	return m.ReloadAllServicesMock(addresses, instanceName, mode, listenerAddress)
}

func (m ReconfigureMock) GetTemplates(sr *proxy.Service) (front, back string, err error) {
	return m.GetTemplatesMock(sr)
}

type CertMock struct {
	PutMock     func(http.ResponseWriter, *http.Request) (string, error)
	PutCertMock func(certName string, certContent []byte) (string, error)
	GetAllMock  func(w http.ResponseWriter, req *http.Request) (CertResponse, error)
	GetInitMock func() error
}

func (m CertMock) Put(w http.ResponseWriter, req *http.Request) (string, error) {
	return m.PutMock(w, req)
}

func (m CertMock) PutCert(certName string, certContent []byte) (string, error) {
	return m.PutCertMock(certName, certContent)
}

func (m CertMock) GetAll(w http.ResponseWriter, req *http.Request) (CertResponse, error) {
	return m.GetAllMock(w, req)
}

func (m CertMock) Init() error {
	return m.GetInitMock()
}