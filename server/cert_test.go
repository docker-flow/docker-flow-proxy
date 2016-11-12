package server

import (
	"../proxy"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type CertTestSuite struct {
	suite.Suite
}

func (s *CertTestSuite) SetupTest() {
}

func TestCertUnitTestSuite(t *testing.T) {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	s := new(CertTestSuite)
	suite.Run(t, s)
}

// GetAll

func (s *CertTestSuite) Test_GetAll_SetsContentTypeToJson() {
	var actual string
	orig := httpWriterSetContentType
	defer func() { httpWriterSetContentType = orig }()
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"GET",
		"http://acme.com/v1/docker-flow-proxy/certs",
		nil,
	)

	c.GetAll(w, req)

	s.Equal("application/json", actual)
}

func (s *CertTestSuite) Test_GetAll_WritesHeaderStatus200() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"GET",
		"http://acme.com/v1/docker-flow-proxy/certs",
		nil,
	)

	c.GetAll(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 200)
}

// NOTE: The assert sometimes fails due to different order of JSON entries.
// TODO: Rewrite tests
func (s *CertTestSuite) Test_GetAll_WritesCertsAsJson() {
	certs := []Cert{}
	proxyCerts := map[string]string{}
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("my-service-%d", i)
		cert := Cert{
			ServiceName: name,
			CertsDir:    "/certs",
			CertContent: fmt.Sprintf("Content of the cert %d", i),
		}
		proxyCerts[name] = fmt.Sprintf("Content of the cert %d", i)
		certs = append(certs, cert)
	}
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("GetCerts")
	proxyMock.On("GetCerts").Return(proxyCerts)
	proxy.Instance = proxyMock
	expected, _ := json.Marshal(CertResponse{
		Status:  "OK",
		Message: "",
		Certs:   certs,
	})
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"GET",
		"http://acme.com/v1/docker-flow-proxy/certs",
		nil,
	)

	c.GetAll(w, req)

	w.AssertCalled(s.T(), "Write", []byte(expected))
}

// Init

//func (s *CertTestSuite) Test_Init_WritesCertsToFiles() {
//	c := NewCert("../certs")
//	certs := map[string]string{}
//	for i := 1; i <= 3; i++ {
//		certName := fmt.Sprintf("my-cert-%d.pem", i)
//		certs[certName] = fmt.Sprintf("Content of my-cert-%s.pem", i)
//		path := fmt.Sprintf("%s/%s", c.CertsDir, certName)
//		os.Remove(path)
//	}
//	proxyOrig := proxy.Instance
//	defer func() { proxy.Instance = proxyOrig }()
//	proxyMock := getProxyMock("GetCerts")
//	proxy.Instance = proxyMock
//	proxyMock.On("GetCerts").Return(certs)
//	expected := "THIS IS A CERTIFICATE"
//
//	c.Init()
//
//	for i := 1; i <= 3; i++ {
//		certName := fmt.Sprintf("my-cert-%d.pem", i)
//		actual, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", c.CertsDir, certName))
//		s.Equal(expected, string(actual))
//	}
//}

//func (s *ServerTestSuite) Test_Init_InvokesLookupHost() {
//	var actualHost string
//	lookupHostOrig := lookupHost
//	defer func() { lookupHost = lookupHostOrig }()
//	lookupHost = func(host string) (addrs []string, err error) {
//		actualHost = host
//		return []string{}, nil
//	}
//	c := NewCert("../certs")
//
//	c.Init()
//
//	s.Assert().Equal(fmt.Sprintf("tasks.%s", s.ServiceName), actualHost)
//}

// Put

func (s *CertTestSuite) Test_Put_SavesBodyAsFile() {
	c := NewCert("../certs")
	certName := "test.pem"
	expected := "THIS IS A CERTIFICATE"
	path := fmt.Sprintf("%s/%s", c.CertsDir, certName)
	os.Remove(path)
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=%s", certName),
		strings.NewReader(expected),
	)

	c.Put(w, req)
	actual, err := ioutil.ReadFile(path)

	s.NoError(err)
	s.Equal(expected, string(actual))
}

func (s *CertTestSuite) Test_Put_InvokesProxyAddCert() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock
	c := NewCert("../certs")
	certName := "test.pem"
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=%s", certName),
		strings.NewReader("THIS IS A CERTIFICATE"),
	)

	c.Put(w, req)

	proxyMock.AssertCalled(s.T(), "AddCert", certName)
}

func (s *CertTestSuite) Test_Put_SetsContentTypeToJson() {
	var actual string
	orig := httpWriterSetContentType
	defer func() { httpWriterSetContentType = orig }()
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem",
		strings.NewReader("cert content"),
	)

	c.Put(w, req)

	s.Equal("application/json", actual)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus200() {
	expected, _ := json.Marshal(CertResponse{
		Status: "OK",
	})
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem",
		strings.NewReader("cert content"),
	)

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 200)
	w.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *CertTestSuite) Test_Put_SendsDistributeRequests_WhenDistruibuteParamIsPresent() {
	serviceName := "my-proxy-service"
	serviceNameOrig := os.Getenv("SERVICE_NAME")
	defer func() { os.Setenv("SERVICE_NAME", serviceNameOrig) }()
	os.Setenv("SERVICE_NAME", serviceName)
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com:1234/v1/docker-flow-proxy/cert?certName=my-cert.pem&distribute=true",
		strings.NewReader("cert content"),
	)
	serverOrig := server
	defer func() { server = serverOrig }()
	mockObj := getServerMock("")
	server = mockObj

	c.Put(w, req)

	mockObj.AssertCalled(s.T(), "SendDistributeRequests", req, "1234", serviceName)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenCertNameIsNotPresent() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com:1234/v1/docker-flow-proxy/cert",
		strings.NewReader("cert content"),
	)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_SendsDistributeRequestsToPort8080_WhenPortIsNotAvailable() {
	serviceName := "my-proxy-service"
	serviceNameOrig := os.Getenv("SERVICE_NAME")
	defer func() { os.Setenv("SERVICE_NAME", serviceNameOrig) }()
	os.Setenv("SERVICE_NAME", serviceName)
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem&distribute=true",
		strings.NewReader("cert content"),
	)
	serverOrig := server
	defer func() { server = serverOrig }()
	mockObj := getServerMock("")
	server = mockObj

	c.Put(w, req)

	mockObj.AssertCalled(s.T(), "SendDistributeRequests", req, "8080", serviceName)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenSendDistributeRequestsReturnsError() {
	serviceName := "my-proxy-service"
	serviceNameOrig := os.Getenv("SERVICE_NAME")
	defer func() { os.Setenv("SERVICE_NAME", serviceNameOrig) }()
	os.Setenv("SERVICE_NAME", serviceName)
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem&distribute=true",
		strings.NewReader("cert content"),
	)
	serverOrig := server
	defer func() { server = serverOrig }()
	mockObj := getServerMock("SendDistributeRequests")
	mockObj.On("SendDistributeRequests", mock.Anything, mock.Anything, mock.Anything).Return(200, fmt.Errorf("This is an error"))
	server = mockObj

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenSendDistributeRequestsReturnsNon200Status() {
	serviceName := "my-proxy-service"
	serviceNameOrig := os.Getenv("SERVICE_NAME")
	defer func() { os.Setenv("SERVICE_NAME", serviceNameOrig) }()
	os.Setenv("SERVICE_NAME", serviceName)
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem&distribute=true",
		strings.NewReader("cert content"),
	)
	serverOrig := server
	defer func() { server = serverOrig }()
	mockObj := getServerMock("SendDistributeRequests")
	mockObj.On("SendDistributeRequests", mock.Anything, mock.Anything, mock.Anything).Return(400, nil)
	server = mockObj

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenDirectoryDoesNotExist() {
	c := NewCert("THIS_PATH_DOES_NOT_EXIST")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem",
		strings.NewReader("cert content"),
	)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus400_WhenDirectoryDoesNotExist() {
	c := NewCert("THIS_PATH_DOES_NOT_EXIST")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem",
		strings.NewReader("cert content"),
	)

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenCannotReadBody() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	r := ReaderMock{
		ReadMock: func([]byte) (int, error) { return 0, fmt.Errorf("This is an error") },
	}
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", r)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus40_WhenCannotReadBody() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	r := ReaderMock{
		ReadMock: func([]byte) (int, error) { return 0, fmt.Errorf("This is an error") },
	}
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", r)

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *CertTestSuite) Test_Put_ReturnsCertPath() {
	c := NewCert("../certs")
	certName := "test.pem"
	expected, _ := filepath.Abs(fmt.Sprintf("%s/%s", c.CertsDir, certName))
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=%s", certName),
		strings.NewReader("cert content"),
	)

	actual, _ := c.Put(w, req)

	s.Equal(expected, actual)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenCertNameDoesNotExist() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert"),
		strings.NewReader("cert content"),
	)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenBodyIsEmpty() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem"),
		strings.NewReader(""),
	)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_InvokesProxyCreateConfigFromTemplates() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem",
		strings.NewReader("cert content"),
	)
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock

	c.Put(w, req)

	proxyMock.AssertCalled(s.T(), "CreateConfigFromTemplates")
}

// DistributeAll

//func (s *CertTestSuite) Test_DistributeAll_ReturnsTheListOfAllCertificates() {
//	expected := map[string]bool{"my-cert-1": true, "my-cert-2": true}
//	c := NewCert("../certs")
//	certsOrig := proxy.Data.Certs
//	defer func(){ proxy.Data.Certs = certsOrig }()
//	proxy.Data
//}

// NewCert

func (s *CertTestSuite) Test_NewCert_SetsCertsDir() {
	expected := "../certs"
	cert := NewCert(expected)

	s.Equal(expected, cert.CertsDir)
}

func (s *CertTestSuite) Test_NewCert_SetsServiceName() {
	serviceName := "my-proxy-service"
	serviceNameOrig := os.Getenv("SERVICE_NAME")
	defer func() { os.Setenv("SERVICE_NAME", serviceNameOrig) }()
	os.Setenv("SERVICE_NAME", serviceName)

	cert := NewCert("../certs")

	s.Equal(serviceName, cert.ServiceName)
}

// Mock

// ReaderMock

type ReaderMock struct {
	ReadMock func([]byte) (int, error)
}

func (m ReaderMock) Read(p []byte) (int, error) {
	return m.ReadMock(p)
}

// ResponseWriterMock

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

type ProxyMock struct {
	mock.Mock
}

func (m *ProxyMock) RunCmd(extraArgs []string) error {
	params := m.Called(extraArgs)
	return params.Error(0)
}

func (m *ProxyMock) CreateConfigFromTemplates() error {
	params := m.Called()
	return params.Error(0)
}

func (m *ProxyMock) ReadConfig() (string, error) {
	params := m.Called()
	return params.String(0), params.Error(1)
}

func (m *ProxyMock) Reload() error {
	params := m.Called()
	return params.Error(0)
}

func (m *ProxyMock) AddCert(certName string) {
	m.Called(certName)
}

func (m *ProxyMock) GetCerts() map[string]string {
	params := m.Called()
	return params.Get(0).(map[string]string)
}

func getProxyMock(skipMethod string) *ProxyMock {
	mockObj := new(ProxyMock)
	if skipMethod != "RunCmd" {
		mockObj.On("RunCmd", mock.Anything).Return(nil)
	}
	if skipMethod != "CreateConfigFromTemplates" {
		mockObj.On("CreateConfigFromTemplates").Return(nil)
	}
	if skipMethod != "ReadConfig" {
		mockObj.On("ReadConfig").Return("", nil)
	}
	if skipMethod != "Reload" {
		mockObj.On("Reload").Return(nil)
	}
	if skipMethod != "AddCert" {
		mockObj.On("AddCert", mock.Anything).Return(nil)
	}
	if skipMethod != "GetCerts" {
		mockObj.On("GetCerts").Return(map[string]string{})
	}
	return mockObj
}
