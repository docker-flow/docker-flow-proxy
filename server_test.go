// +build !integration

package main

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

type ServerTestSuite struct {
	suite.Suite
	ServiceReconfigure
	ConsulAddress      string
	ReconfigureBaseUrl string
	RemoveBaseUrl      string
	ReconfigureUrl     string
	RemoveUrl          string
	ConfigUrl          string
	ResponseWriter     *ResponseWriterMock
	RequestReconfigure *http.Request
	RequestRemove      *http.Request
	InstanceName       string
	DnsIps             []string
	Server             *httptest.Server
}

func (s *ServerTestSuite) SetupTest() {
	s.InstanceName = "proxy-test-instance"
	s.ConsulAddress = "http://1.2.3.4:1234"
	s.ServiceName = "myService"
	s.ServiceColor = "pink"
	s.ServiceDomain = "my-domain.com"
	s.ServicePath = []string{"/path/to/my/service/api", "/path/to/my/other/service/api"}
	s.OutboundHostname = "machine-123.my-company.com"
	s.ReconfigureBaseUrl = "/v1/docker-flow-proxy/reconfigure"
	s.RemoveBaseUrl = "/v1/docker-flow-proxy/remove"
	s.ReconfigureUrl = fmt.Sprintf(
		"%s?serviceName=%s&serviceColor=%s&servicePath=%s&serviceDomain=%s&outboundHostname=%s",
		s.ReconfigureBaseUrl,
		s.ServiceName,
		s.ServiceColor,
		strings.Join(s.ServicePath, ","),
		s.ServiceDomain,
		s.OutboundHostname,
	)
	s.RemoveUrl = fmt.Sprintf(
		"%s?serviceName=%s",
		s.RemoveBaseUrl,
		s.ServiceName,
	)
	s.ConfigUrl = "/v1/docker-flow-proxy/config"
	s.ResponseWriter = getResponseWriterMock()
	s.RequestReconfigure, _ = http.NewRequest("GET", s.ReconfigureUrl, nil)
	s.RequestRemove, _ = http.NewRequest("GET", s.RemoveUrl, nil)
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return nil
	}
	server = Serve{
		BaseReconfigure: BaseReconfigure{
			ConsulAddresses: []string{s.ConsulAddress},
			InstanceName:    s.InstanceName,
		},
	}
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return getReconfigureMock("")
	}
	logPrintf = func(format string, v ...interface{}) {}
}

// Execute

func (s *ServerTestSuite) Test_Execute_InvokesHTTPListenAndServe() {
	server := Serve{
		IP:   "myIp",
		Port: "1234",
	}
	var actual string
	expected := fmt.Sprintf("%s:%s", server.IP, server.Port)
	httpListenAndServe = func(addr string, handler http.Handler) error {
		actual = addr
		return nil
	}

	server.Execute([]string{})

	s.Equal(expected, actual)
}

func (s *ServerTestSuite) Test_Execute_ReturnsError_WhenHTTPListenAndServeFails() {
	orig := httpListenAndServe
	defer func() {
		httpListenAndServe = orig
	}()
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return fmt.Errorf("This is an error")
	}

	actual := server.Execute([]string{})

	s.Error(actual)
}

func (s *ServerTestSuite) Test_Execute_InvokesRunExecute() {
	orig := NewRun
	defer func() {
		NewRun = orig
	}()
	mockObj := getRunMock("")
	NewRun = func() Executable {
		return mockObj
	}

	server.Execute([]string{})

	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

func (s *ServerTestSuite) Test_Execute_InvokesReloadAllServices() {
	mockObj := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}
	consulAddressesOrig := []string{s.ConsulAddress}
	defer func() {
		os.Unsetenv("CONSUL_ADDRESS")
		server.ConsulAddresses = consulAddressesOrig
	}()
	os.Setenv("CONSUL_ADDRESS", s.ConsulAddress)

	server.Execute([]string{})

	mockObj.AssertCalled(s.T(), "ReloadAllServices", []string{s.ConsulAddress}, s.InstanceName, s.Mode, "")
}

func (s *ServerTestSuite) Test_Execute_InvokesReloadAllServicesWithListenerAddress() {
	listenerAddress := "swarm-listener"
	mockObj := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}
	consulAddressesOrig := []string{s.ConsulAddress}
	defer func() {
		os.Unsetenv("CONSUL_ADDRESS")
		os.Unsetenv("LISTENER_ADDRESS")
		server.ConsulAddresses = consulAddressesOrig
	}()
	os.Setenv("CONSUL_ADDRESS", s.ConsulAddress)
	server.ListenerAddress = listenerAddress

	server.Execute([]string{})

	mockObj.AssertCalled(
		s.T(),
		"ReloadAllServices",
		[]string{s.ConsulAddress},
		s.InstanceName,
		s.Mode,
		fmt.Sprintf("http://%s:8080", listenerAddress),
	)
}

func (s *ServerTestSuite) Test_Execute_DoesNotInvokeReloadAllServices_WhenModeIsService() {
	server.Mode = "seRviCe"
	mockObj := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}

	server.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "ReloadAllServices", s.ConsulAddress, s.InstanceName, s.Mode)
}

func (s *ServerTestSuite) Test_Execute_DoesNotInvokeReloadAllServices_WhenModeIsSwarm() {
	server.Mode = "SWarM"
	mockObj := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}

	server.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "ReloadAllServices", s.ConsulAddress, s.InstanceName, s.Mode)
}

func (s *ServerTestSuite) Test_Execute_ReturnsError_WhenReloadAllServicesFails() {
	mockObj := getReconfigureMock("ReloadAllServices")
	mockObj.On("ReloadAllServices", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}

	actual := server.Execute([]string{})

	s.Error(actual)
}

func (s *ServerTestSuite) Test_Execute_SetsConsulAddressesToEmptySlice_WhenEnvVarIsNotset() {
	srv := Serve{}

	srv.Execute([]string{})

	s.Equal([]string{}, srv.ConsulAddresses)
}

func (s *ServerTestSuite) Test_Execute_SetsConsulAddresses() {
	expected := "http://my-consul"
	consulAddressesOrig := server.ConsulAddresses
	defer func() {
		os.Unsetenv("CONSUL_ADDRESS")
		server.ConsulAddresses = consulAddressesOrig
	}()
	os.Setenv("CONSUL_ADDRESS", expected)
	srv := Serve{}

	srv.Execute([]string{})

	s.Equal([]string{expected}, srv.ConsulAddresses)
}

func (s *ServerTestSuite) Test_Execute_SetsMultipleConsulAddresseses() {
	expected := []string{"http://my-consul-1", "http://my-consul-2"}
	consulAddressesOrig := server.ConsulAddresses
	defer func() {
		os.Unsetenv("CONSUL_ADDRESS")
		server.ConsulAddresses = consulAddressesOrig
	}()
	os.Setenv("CONSUL_ADDRESS", strings.Join(expected, ","))
	srv := Serve{}

	srv.Execute([]string{})

	s.Equal(expected, srv.ConsulAddresses)
}

func (s *ServerTestSuite) Test_Execute_AddsHttpToConsulAddresses() {
	expected := []string{"http://my-consul-1", "http://my-consul-2"}
	consulAddressesOrig := server.ConsulAddresses
	defer func() {
		os.Unsetenv("CONSUL_ADDRESS")
		server.ConsulAddresses = consulAddressesOrig
	}()
	os.Setenv("CONSUL_ADDRESS", "my-consul-1,my-consul-2")
	srv := Serve{}

	srv.Execute([]string{})

	s.Equal(expected, srv.ConsulAddresses)
}

// ServeHTTP

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus404WhenURLIsUnknown() {
	req, _ := http.NewRequest("GET", "/this/url/does/not/exist", nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 404)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus200WhenUrlIsTest() {
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/v%d/test", ver), nil)

		srv := Serve{}
		srv.ServeHTTP(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

// ServeHTTP > Reconfigure

func (s *ServerTestSuite) Test_ServeHTTP_SetsContentTypeToJSON_WhenUrlIsReconfigure() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJSON_WhenUrlIsReconfigure() {
	expected, _ := json.Marshal(Response{
		Status:           "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		PathType:         s.PathType,
	})

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, s.RequestReconfigure)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJsonWithPathType_WhenPresent() {
	pathType := "path_reg"
	req, _ := http.NewRequest("GET", s.ReconfigureUrl+"&pathType="+pathType, nil)
	expected, _ := json.Marshal(Response{
		Status:           "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		PathType:         pathType,
	})

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJsonWithPort_WhenPresent() {
	port := "1234"
	mode := "swaRM"
	req, _ := http.NewRequest("GET", s.ReconfigureUrl+"&port="+port, nil)
	expected, _ := json.Marshal(Response{
		Status:        "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		Port:             port,
		Mode:             mode,
	})

	srv := Serve{Mode: mode}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJsonWithSkipCheck_WhenPresent() {
	req, _ := http.NewRequest("GET", s.ReconfigureUrl+"&skipCheck=true", nil)
	expected, _ := json.Marshal(Response{
		Status:        "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		PathType:         s.PathType,
		SkipCheck:        true,
	})

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJsonWithDistribute_WhenReconfigureAndDistributePresent() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true", s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)
	expected, _ := json.Marshal(Response{
		Status:        "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		PathType:         s.PathType,
		Distribute:       true,
		Message:          DISTRIBUTED,
	})

	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJsonWithDistribute_WhenRemoveAndDistributePresent() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true", s.RemoveUrl)
	req, _ := http.NewRequest("GET", addr, nil)
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: s.ServiceName,
		Distribute:  true,
		Message:     DISTRIBUTED,
	})

	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_WritesDistributed_WhenReconfigureAndDistributeIsTrue() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true", s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)
	expected, _ := json.Marshal(Response{
		Status:           "OK",
		ServiceName:      s.ServiceName,
		ServiceColor:     s.ServiceColor,
		ServicePath:      s.ServicePath,
		ServiceDomain:    s.ServiceDomain,
		OutboundHostname: s.OutboundHostname,
		PathType:         s.PathType,
		Distribute:       true,
		Message:          DISTRIBUTED,
	})
	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_WritesDistributed_WhenRemoveAndDistributeIsTrue() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true", s.RemoveUrl)
	req, _ := http.NewRequest("GET", addr, nil)
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: s.ServiceName,
		Distribute:  true,
		Message:     DISTRIBUTED,
	})

	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_WritesErrorHeader_WhenReconfigureDistributeIsTrueAndError() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true&returnError=true", s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ServeHTTP_WritesErrorHeader_WhenRemoveDistributeIsTrueAndError() {
	serve := Serve{}
	serve.Port = s.Port
	addr := fmt.Sprintf("http://127.0.0.1:8080%s&distribute=true&returnError=true", s.RemoveUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	serve.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus400_WhenUrlIsReconfigureAndServiceNameQueryIsNotPresent() {
	req, _ := http.NewRequest("GET", s.ReconfigureBaseUrl, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus400_WhenServicePathQueryIsNotPresent() {
	url := fmt.Sprintf("%s?serviceName=%s", s.ReconfigureBaseUrl, s.ServiceName[0])
	req, _ := http.NewRequest("GET", url, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus400_WhenModeIsServiceAndPortIsNotPresent() {
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{Mode: "service"}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus400_WhenModeIsSwarmAndPortIsNotPresent() {
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{Mode: "swARM"}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ServeHTTP_InvokesReconfigureExecute() {
	s.invokesReconfigure(s.RequestReconfigure, true)
}

func (s *ServerTestSuite) Test_ServeHTTP_DoesNotInvokeReconfigureExecute_WhenDistributeIsTrue() {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s&distribute=true", s.ReconfigureUrl),
		nil,
	)
	s.invokesReconfigure(req, false)
}

func (s *ServerTestSuite) Test_ServeHTTP_DoesNotInvokeRemoveExecute_WhenDistributeIsTrue() {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s&distribute=true", s.RemoveUrl),
		nil,
	)
	s.invokesReconfigure(req, false)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus500_WhenReconfigureExecuteFails() {
	mockObj := getReconfigureMock("Execute")
	mockObj.On("Execute", []string{}).Return(fmt.Errorf("This is an error"))
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		return mockObj
	}

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, s.RequestReconfigure)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJson_WhenConsulTemplatePathIsPresent() {
	pathFe := "/path/to/consul/fe/template"
	pathBe := "/path/to/consul/fe/template"
	address := fmt.Sprintf(
		"%s?serviceName=%s&consulTemplateFePath=%s&consulTemplateBePath=%s",
		s.ReconfigureBaseUrl,
		s.ServiceName,
		pathFe,
		pathBe)
	req, _ := http.NewRequest("GET", address, nil)
	expected, _ := json.Marshal(Response{
		Status:               "OK",
		ServiceName:          s.ServiceName,
		ConsulTemplateFePath: pathFe,
		ConsulTemplateBePath: pathBe,
		PathType:             s.PathType,
	})

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_InvokesReconfigureExecute_WhenConsulTemplatePathIsPresent() {
	pathFe := "/path/to/consul/fe/template"
	pathBe := "/path/to/consul/be/template"
	mockObj := getReconfigureMock("")
	var actualBase BaseReconfigure
	expectedBase := BaseReconfigure{
		ConsulAddresses: []string{s.ConsulAddress},
	}
	expectedService := ServiceReconfigure{
		ServiceName:          s.ServiceName,
		ConsulTemplateFePath: pathFe,
		ConsulTemplateBePath: pathBe,
		PathType:             s.PathType,
	}
	var actualService ServiceReconfigure
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		actualBase = baseData
		actualService = serviceData
		return mockObj
	}
	server := Serve{BaseReconfigure: expectedBase}
	address := fmt.Sprintf(
		"%s?serviceName=%s&consulTemplateFePath=%s&consulTemplateBePath=%s",
		s.ReconfigureBaseUrl,
		s.ServiceName,
		pathFe,
		pathBe)
	req, _ := http.NewRequest("GET", address, nil)

	server.ServeHTTP(s.ResponseWriter, req)

	s.Equal(expectedBase, actualBase)
	s.Equal(expectedService, actualService)
	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

// ServeHTTP > Remove

func (s *ServerTestSuite) Test_ServeHTTP_SetsContentTypeToJSON_WhenUrlIsRemove() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", s.RemoveUrl, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsJSON_WhenUrlIsRemove() {
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: s.ServiceName,
	})

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, s.RequestRemove)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus400_WhenUrlIsRemoveAndServiceNameQueryIsNotPresent() {
	req, _ := http.NewRequest("GET", s.RemoveBaseUrl, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ServeHTTP_InvokesRemoveExecute() {
	mockObj := getRemoveMock("")
	var actual Remove
	expected := Remove{
		ServiceName:     s.ServiceName,
		TemplatesPath:   "",
		ConfigsPath:     "",
		ConsulAddresses: []string{s.ConsulAddress},
		InstanceName:    s.InstanceName,
	}
	NewRemove = func(serviceName, configsPath, templatesPath string, consulAddresses []string, instanceName, mode string) Removable {
		actual = Remove{
			ServiceName:     serviceName,
			TemplatesPath:   templatesPath,
			ConfigsPath:     configsPath,
			ConsulAddresses: consulAddresses,
			InstanceName:    instanceName,
			Mode:            mode,
		}
		return mockObj
	}

	server.ServeHTTP(s.ResponseWriter, s.RequestRemove)

	s.Equal(expected, actual)
	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

// ServeHTTP > Config

func (s *ServerTestSuite) Test_ServeHTTP_SetsContentTypeToText_WhenUrlIsConfig() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", s.ConfigUrl, nil)

	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.Equal("text/html", actual)
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsConfig_WhenUrlIsConfig() {
	expected := "some text"
	readFileOrig := readFile
	defer func() { readFile = readFileOrig }()
	readFile = func(filename string) ([]byte, error) {
		return []byte(expected), nil
	}

	req, _ := http.NewRequest("GET", s.ConfigUrl, nil)
	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_ServeHTTP_ReturnsStatus500_WhenReadFileFails() {
	readFileOrig := readFile
	defer func() { readFile = readFileOrig }()
	readFile = func(filename string) ([]byte, error) {
		return []byte(""), fmt.Errorf("This is an error")
	}

	req, _ := http.NewRequest("GET", s.ConfigUrl, nil)
	srv := Serve{}
	srv.ServeHTTP(s.ResponseWriter, req)

	s.ResponseWriter.AssertCalled(s.T(), "WriteHeader", 500)
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
	server.ServiceName = "my-fancy-proxy"

	server.SendDistributeRequests(req, s.ServiceName)

	s.Assert().Equal(fmt.Sprintf("tasks.%s", server.ServiceName), actualHost)
}

func (s *ServerTestSuite) Test_SednDistributeRequests_ReturnsError_WhenLookupHostFails() {
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return []string{}, fmt.Errorf("This is an LookupHost error")
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	status, err := server.SendDistributeRequests(req, s.ServiceName)

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
	portOrig := server.Port
	defer func() { server.Port = portOrig }()
	server.Port = strings.Split(tsAddr, ":")[1]

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", server.Port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	server.SendDistributeRequests(req, s.ServiceName)

	s.Assert().Equal(s.ReconfigureBaseUrl, actualPath)
	s.Assert().Equal("false", actualQuery.Get("distribute"))
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
	portOrig := server.Port
	defer func() { server.Port = portOrig }()
	server.Port = strings.Split(tsAddr, ":")[1]

	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", server.Port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	status, err := server.SendDistributeRequests(req, s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

// Suite

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	logPrintf = func(format string, v ...interface{}) {}
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
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
	addr := strings.Replace(s.Server.URL, "http://", "", -1)
	s.DnsIps = []string{strings.Split(addr, ":")[0]}
	lookupHost = func(host string) (addrs []string, err error) {
		return s.DnsIps, nil
	}
	s.Port = strings.Split(addr, ":")[1]

	suite.Run(t, s)
}

// Mock

type ServerMock struct {
	mock.Mock
}

func (m *ServerMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func (m *ServerMock) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.Called(w, req)
}

func getServerMock() *ServerMock {
	mockObj := new(ServerMock)
	mockObj.On("Execute", mock.Anything).Return(nil)
	mockObj.On("ServeHTTP", mock.Anything, mock.Anything)
	return mockObj
}

type ResponseWriterMock struct {
	mock.Mock
}

func (m *ResponseWriterMock) Header() http.Header {
	m.Called()
	return make(map[string][]string)
}

func (m *ResponseWriterMock) Write(data []byte) (int, error) {
	params := m.Called(data)
	return params.Int(0), params.Error(1)
}

func (m *ResponseWriterMock) WriteHeader(header int) {
	m.Called(header)
}

func getResponseWriterMock() *ResponseWriterMock {
	mockObj := new(ResponseWriterMock)
	mockObj.On("Header").Return(nil)
	mockObj.On("Write", mock.Anything).Return(0, nil)
	mockObj.On("WriteHeader", mock.Anything)
	return mockObj
}

// Util

func (s *ServerTestSuite) invokesReconfigure(req *http.Request, invoke bool) {
	mockObj := getReconfigureMock("")
	var actualBase BaseReconfigure
	expectedBase := BaseReconfigure{
		ConsulAddresses: []string{s.ConsulAddress},
	}
	var actualService ServiceReconfigure
	NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
		actualBase = baseData
		actualService = serviceData
		return mockObj
	}
	server := Serve{BaseReconfigure: expectedBase}
	portOrig := s.Port
	defer func() { s.Port = portOrig }()
	s.Port = ""

	server.ServeHTTP(s.ResponseWriter, req)

	if invoke {
		s.Equal(expectedBase, actualBase)
		s.Equal(s.ServiceReconfigure, actualService)
		mockObj.AssertCalled(s.T(), "Execute", []string{})
	} else {
		mockObj.AssertNotCalled(s.T(), "Execute", []string{})
	}
}
