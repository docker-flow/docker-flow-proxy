// +build !integration

package main

import (
	"./registry"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type ReconfigureTestSuite struct {
	suite.Suite
	ServiceReconfigure
	ConsulAddress     string
	ConsulTemplateFe  string
	ConsulTemplateBe  string
	ConfigsPath       string
	TemplatesPath     string
	reconfigure       Reconfigure
	Pid               string
	Server            *httptest.Server
	PutPathResponse   string
	ConsulRequestBody ServiceReconfigure
	InstanceName      string
	SkipCheck         bool
}

func (s *ReconfigureTestSuite) SetupTest() {
	s.InstanceName = "proxy-test-instance"
	s.Pid = "123"
	s.ServicePath = []string{"path/to/my/service/api", "path/to/my/other/service/api"}
	s.ConfigsPath = "path/to/configs/dir"
	s.TemplatesPath = "test_configs/tmpl"
	s.SkipCheck = false
	s.ConsulTemplateFe = `
    acl url_myService path_beg path/to/my/service/api path_beg path/to/my/other/service/api
    use_backend myService-be if url_myService`
	s.ConsulTemplateBe = `backend myService-be
    {{range $i, $e := service "myService" "any"}}
    server {{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check
    {{end}}`
	cmdRunHa = func(cmd *exec.Cmd) error {
		return nil
	}
	readPidFile = func(fileName string) ([]byte, error) {
		return []byte(s.Pid), nil
	}
	s.ConsulAddress = s.Server.URL
	s.reconfigure = Reconfigure{
		BaseReconfigure: BaseReconfigure{
			ConsulAddresses: []string{s.ConsulAddress},
			TemplatesPath:   s.TemplatesPath,
			ConfigsPath:     s.ConfigsPath,
			InstanceName:    s.InstanceName,
		},
		ServiceReconfigure: ServiceReconfigure{
			ServiceName: s.ServiceName,
			ServicePath: s.ServicePath,
			PathType:    s.PathType,
			SkipCheck:   false,
		},
	}
	s.reconfigure.skipAddressValidation = true
}

// GetConsulTemplate

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent() {
	front, back, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplateFe, front)
	s.Equal(s.ConsulTemplateBe, back)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent_WhenModeIsService() {
	s.reconfigure.ServiceReconfigure.Mode = "service"
	s.reconfigure.ServiceReconfigure.Port = "1234"
	expected := `backend myService-be
    server myService myService:1234`

	_, actual, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent_WhenModeIsSwarm() {
	s.reconfigure.ServiceReconfigure.Mode = "sWARm"
	s.reconfigure.ServiceReconfigure.Port = "1234"
	expected := `backend myService-be
    server myService myService:1234`

	_, actual, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_AddsHost() {
	s.ConsulTemplateFe = `
    acl url_myService path_beg path/to/my/service/api path_beg path/to/my/other/service/api
    acl domain_myService hdr_dom(host) -i my-domain.com
    use_backend myService-be if url_myService domain_myService`
	s.reconfigure.ServiceDomain = "my-domain.com"
	actual, _, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplateFe, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_UsesPathReg() {
	s.ConsulTemplateFe = strings.Replace(s.ConsulTemplateFe, "path_beg", "path_reg", -1)
	s.reconfigure.PathType = "path_reg"
	front, _, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplateFe, front)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_AddsColor() {
	s.reconfigure.ServiceColor = "black"
	expected := fmt.Sprintf(`service "%s-%s"`, s.ServiceName, s.reconfigure.ServiceColor)

	_, actual, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Contains(actual, expected)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_DoesNotSetCheckWhenSkipCheckIsTrue() {
	s.ConsulTemplateBe = strings.Replace(s.ConsulTemplateBe, " check", "", -1)
	s.reconfigure.SkipCheck = true
	_, actual, _ := s.reconfigure.GetTemplates(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplateBe, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFileContent_WhenConsulTemplatePathIsSet() {
	expected := "This is content of a template"
	readTemplateFileOrig := readTemplateFile
	defer func() { readTemplateFile = readTemplateFileOrig }()
	readTemplateFile = func(dirname string) ([]byte, error) {
		return []byte(expected), nil
	}
	s.ServiceReconfigure.ConsulTemplateFePath = "/path/to/my/consul/fe/template"
	s.ServiceReconfigure.ConsulTemplateBePath = "/path/to/my/consul/be/template"

	_, actual, _ := s.reconfigure.GetTemplates(s.ServiceReconfigure)

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsError_WhenConsulTemplateFileIsNotAvailable() {
	readTemplateFileOrig := readTemplateFile
	defer func() { readTemplateFile = readTemplateFileOrig }()
	readTemplateFile = func(dirname string) ([]byte, error) {
		return nil, fmt.Errorf("This is an error")
	}
	s.ServiceReconfigure.ConsulTemplateFePath = "/path/to/my/consul/fe/template"
	s.ServiceReconfigure.ConsulTemplateBePath = "/path/to/my/consul/be/template"

	_, _, actual := s.reconfigure.GetTemplates(s.ServiceReconfigure)

	s.Error(actual)
}

// Execute

func (s ReconfigureTestSuite) Test_Execute_InvokesRegistrarableCreateConfigs() {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	expectedArgs := registry.CreateConfigsArgs{
		Addresses:     []string{s.ConsulAddress},
		TemplatesPath: s.TemplatesPath,
		FeFile:        ServiceTemplateFeFilename,
		FeTemplate:    s.ConsulTemplateFe,
		BeFile:        ServiceTemplateBeFilename,
		BeTemplate:    s.ConsulTemplateBe,
		ServiceName:   s.ServiceName,
	}

	s.reconfigure.Execute([]string{})

	mockObj.AssertCalled(s.T(), "CreateConfigs", &expectedArgs)
}

func (s ReconfigureTestSuite) Test_Execute_WritesFeTemplate_WhenModeIsService() {
	s.reconfigure.Mode = "service"
	var actualFilename, actualData string
	expectedFilename := fmt.Sprintf("%s/%s-fe.cfg", s.TemplatesPath, s.ServiceName)
	writeFeTemplateOrig := writeFeTemplate
	defer func() { writeFeTemplate = writeFeTemplateOrig }()
	writeFeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expectedFilename, actualFilename)
	s.Equal(s.ConsulTemplateFe, actualData)
}

func (s ReconfigureTestSuite) Test_Execute_WritesFeTemplate_WhenModeIsSwarm() {
	s.reconfigure.Mode = "sWarm"
	var actualFilename, actualData string
	expectedFilename := fmt.Sprintf("%s/%s-fe.cfg", s.TemplatesPath, s.ServiceName)
	writeFeTemplateOrig := writeFeTemplate
	defer func() { writeFeTemplate = writeFeTemplateOrig }()
	writeFeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expectedFilename, actualFilename)
	s.Equal(s.ConsulTemplateFe, actualData)
}

func (s ReconfigureTestSuite) Test_Execute_WritesBeTemplate_WhenModeIsService() {
	s.reconfigure.Mode = "SerVIce"
	s.reconfigure.Port = "1234"
	var actualFilename, actualData string
	expectedFilename := fmt.Sprintf("%s/%s-be.cfg", s.TemplatesPath, s.ServiceName)
	expectedData := fmt.Sprintf("backend %s-be\n    server %s %s:%s", s.ServiceName, s.ServiceName, s.ServiceName, s.reconfigure.Port)
	writeBeTemplateOrig := writeBeTemplate
	defer func() { writeBeTemplate = writeBeTemplateOrig }()
	writeBeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s ReconfigureTestSuite) Test_Execute_WritesBeTemplate_WhenModeIsSwarm() {
	s.reconfigure.Mode = "sWArm"
	s.reconfigure.Port = "1234"
	var actualFilename, actualData string
	expectedFilename := fmt.Sprintf("%s/%s-be.cfg", s.TemplatesPath, s.ServiceName)
	expectedData := fmt.Sprintf("backend %s-be\n    server %s %s:%s", s.ServiceName, s.ServiceName, s.ServiceName, s.reconfigure.Port)
	writeBeTemplateOrig := writeBeTemplate
	defer func() { writeBeTemplate = writeBeTemplateOrig }()
	writeBeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s ReconfigureTestSuite) Test_Execute_DoesNotInvokeRegistrarableCreateConfigs_WhenModeIsService() {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	s.reconfigure.Mode = "seRviCe"

	s.reconfigure.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "CreateConfigs", mock.Anything)
}

func (s ReconfigureTestSuite) Test_Execute_DoesNotInvokeRegistrarableCreateConfigs_WhenModeIsSwarm() {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	s.reconfigure.Mode = "sWaRm"

	s.reconfigure.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "CreateConfigs", mock.Anything)
}

func (s ReconfigureTestSuite) Test_Execute_ReturnsError_WhenRegistrarableCreateConfigsFails() {
	mockObj := getRegistrarableMock("CreateConfigs")
	mockObj.On(
		"CreateConfigs",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(fmt.Errorf("This is an error"))
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj

	actual := s.reconfigure.Execute([]string{})

	s.Error(actual)
}

func (s ReconfigureTestSuite) Test_Execute_InvokesProxyCreateConfigFromTemplates() {
	mockObj := getProxyMock("")
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	s.reconfigure.Execute([]string{})

	mockObj.AssertCalled(s.T(), "CreateConfigFromTemplates", s.TemplatesPath, s.ConfigsPath)
}

func (s ReconfigureTestSuite) Test_Execute_ReturnsError_WhenProxyFails() {
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	err := s.reconfigure.Execute([]string{})

	s.Error(err)
}

func (s ReconfigureTestSuite) Test_Execute_InvokesHaProxyReload() {
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	mock := getProxyMock("")
	proxy = mock

	s.reconfigure.Execute([]string{})

	mock.AssertCalled(s.T(), "Reload")
}

func (s *ReconfigureTestSuite) Test_Execute_PutsDataToConsul() {
	s.SkipCheck = true
	s.reconfigure.SkipCheck = true
	s.reconfigure.ServiceDomain = s.ServiceDomain
	s.reconfigure.ConsulTemplateFePath = s.ConsulTemplateFePath
	s.reconfigure.ConsulTemplateBePath = s.ConsulTemplateBePath
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	r := registry.Registry{
		ServiceName:          s.ServiceName,
		ServiceColor:         s.ServiceColor,
		ServicePath:          s.ServicePath,
		ServiceDomain:        s.ServiceDomain,
		OutboundHostname:     s.OutboundHostname,
		PathType:             s.PathType,
		SkipCheck:            s.SkipCheck,
		ConsulTemplateFePath: s.ConsulTemplateFePath,
		ConsulTemplateBePath: s.ConsulTemplateBePath,
	}

	s.reconfigure.Execute([]string{})

	mockObj.AssertCalled(s.T(), "PutService", []string{s.ConsulAddress}, s.InstanceName, r)
}

func (s *ReconfigureTestSuite) Test_Execute_DoesNotPutDataToConsul_WhenModeIsServiceAndConsulAddressIsEmpty() {
	s.verifyDoesNotPutDataToConsul("seRViCe")
}

func (s *ReconfigureTestSuite) Test_Execute_DoesNotPutDataToConsul_WhenModeIsSwarmAndConsulAddressIsEmpty() {
	s.verifyDoesNotPutDataToConsul("SWARm")
}

func (s *ReconfigureTestSuite) Test_Execute_ReturnsError_WhenPutToConsulFails() {
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	mockObj := getRegistrarableMock("PutService")
	mockObj.On("PutService", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	registryInstance = mockObj

	actual := s.reconfigure.Execute([]string{})

	s.Error(actual)
}

func (s *ReconfigureTestSuite) Test_Execute_AddsHttpIfNotPresentInPutToConsul() {
	s.reconfigure.ConsulAddresses = []string{strings.Replace(s.ConsulAddress, "http://", "", -1)}
	s.reconfigure.Execute([]string{})

	s.Equal(s.ServiceColor, s.ConsulRequestBody.ServiceColor)
}

func (s *ReconfigureTestSuite) Test_Execute_SendsServicePathToConsul() {
	s.reconfigure.Execute([]string{})

	s.Equal(s.reconfigure.ServiceColor, s.ConsulRequestBody.ServiceColor)
}

func (s *ReconfigureTestSuite) Test_Execute_ReturnsError_WhenConsulTemplateFileIsNotAvailable() {
	readTemplateFileOrig := readTemplateFile
	defer func() { readTemplateFile = readTemplateFileOrig }()
	readTemplateFile = func(dirname string) ([]byte, error) {
		return nil, fmt.Errorf("This is an error")
	}
	s.reconfigure.ServiceReconfigure.ConsulTemplateFePath = "/path/to/my/consul/fe/template"
	s.reconfigure.ServiceReconfigure.ConsulTemplateBePath = "/path/to/my/consul/be/template"

	err := s.reconfigure.Execute([]string{})

	s.Error(err)
}

func (s *ReconfigureTestSuite) Test_Execute_ReturnsError_WhenAddressIsNotAccessible() {
	s.reconfigure.Mode = "swarm"
	s.reconfigure.ServiceName = "this-service-does-not-exist"
	skipAddressValidationOrig := s.reconfigure.skipAddressValidation
	defer func() { s.reconfigure.skipAddressValidation = skipAddressValidationOrig }()
	s.reconfigure.skipAddressValidation = false
	println("xxx")

	err := s.reconfigure.Execute([]string{})

	s.Error(err)
//	s.NoError(err)
}

// NewReconfigure

func (s *ReconfigureTestSuite) Test_NewReconfigure_AddsBaseAndService() {
	br := BaseReconfigure{ConsulAddresses: []string{"myConsulAddress"}}
	sr := ServiceReconfigure{ServiceName: "myService"}

	r := NewReconfigure(br, sr)

	actualBr, actualSr := r.GetData()
	s.Equal(br, actualBr)
	s.Equal(sr, actualSr)
}

func (s *ReconfigureTestSuite) Test_NewReconfigure_CreatesNewStruct() {
	r1 := NewReconfigure(
		BaseReconfigure{ConsulAddresses: []string{"myConsulAddress"}},
		ServiceReconfigure{ServiceName: "myService"},
	)
	r2 := NewReconfigure(BaseReconfigure{}, ServiceReconfigure{})

	actualBr1, actualSr1 := r1.GetData()
	actualBr2, actualSr2 := r2.GetData()
	s.NotEqual(actualBr1, actualBr2)
	s.NotEqual(actualSr1, actualSr2)
}

// ReloadAllServices

func (s *ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenFail() {
	err := s.reconfigure.ReloadAllServices([]string{"this/address/does/not/exist"}, s.InstanceName, s.Mode, "")

	s.Error(err)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_WritesTemplateToFile() {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	s.ConsulTemplateBe = `backend myService-be
    {{range $i, $e := service "myService-orange" "any"}}
    server {{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check
    {{end}}`
	expectedArgs := registry.CreateConfigsArgs{
		Addresses:     []string{s.Server.URL},
		TemplatesPath: s.TemplatesPath,
		FeFile:        ServiceTemplateFeFilename,
		FeTemplate:    s.ConsulTemplateFe,
		BeFile:        ServiceTemplateBeFilename,
		BeTemplate:    s.ConsulTemplateBe,
		ServiceName:   s.ServiceName,
	}

	s.reconfigure.ReloadAllServices([]string{s.ConsulAddress}, s.InstanceName, s.Mode, "")

	mockObj.AssertCalled(s.T(), "CreateConfigs", &expectedArgs)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_InvokesProxyCreateConfigFromTemplates() {
	mockObj := getProxyMock("")
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	s.reconfigure.ReloadAllServices([]string{s.ConsulAddress}, s.InstanceName, s.Mode, "")

	mockObj.AssertCalled(s.T(), "CreateConfigFromTemplates", s.TemplatesPath, s.ConfigsPath)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenProxyCreateConfigFromTemplatesFails() {
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	actual := s.reconfigure.ReloadAllServices([]string{s.ConsulAddress}, s.InstanceName, s.Mode, "")

	s.Error(actual)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_InvokesProxyReload() {
	mockObj := getProxyMock("")
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	s.reconfigure.ReloadAllServices([]string{s.ConsulAddress}, s.InstanceName, s.Mode, "")

	mockObj.AssertCalled(s.T(), "Reload")
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenProxyReloadFails() {
	mockObj := getProxyMock("Reload")
	mockObj.On("Reload").Return(fmt.Errorf("This is an error"))
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = mockObj

	actual := s.reconfigure.ReloadAllServices([]string{s.ConsulAddress}, s.InstanceName, s.Mode, "")

	s.Error(actual)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_AddsHttpIfNotPresent() {
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	proxy = getProxyMock("")
	address := strings.Replace(s.ConsulAddress, "http://", "", -1)
	err := s.reconfigure.ReloadAllServices([]string{address}, s.InstanceName, s.Mode, "")

	s.NoError(err)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_SendsARequestToSwarmListener_WhenListenerAddressIsDefined() {
	actual := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual = r.URL.Path
	}))
	defer func() { srv.Close() }()

	s.reconfigure.ReloadAllServices([]string{}, s.InstanceName, s.Mode, srv.URL)

	s.Equal("/v1/docker-flow-swarm-listener/notify-services", actual)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenSwarmListenerStatusIsNot200() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer func() { srv.Close() }()

	err := s.reconfigure.ReloadAllServices([]string{}, s.InstanceName, s.Mode, srv.URL)

	s.Error(err)
}

func (s *ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenSwarmListenerFails() {
	httpGetOrig := httpGet
	defer func() { httpGet = httpGetOrig }()
	httpGet = func(url string) (*http.Response, error) {
		resp := http.Response{
			StatusCode: http.StatusOK,
		}
		return &resp, fmt.Errorf("This is an error")
	}
	err := s.reconfigure.ReloadAllServices([]string{}, s.InstanceName, s.Mode, "http://google.com")

	s.Error(err)
}

// Suite

func TestReconfigureUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	s := new(ReconfigureTestSuite)
	s.ServiceName = "myService"
	s.PutPathResponse = "PUT_PATH_OK"
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/catalog/services":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				data := map[string][]string{"service1": []string{}, "service2": []string{}, s.ServiceName: []string{}}
				js, _ := json.Marshal(data)
				w.Write(js)
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.PATH_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(strings.Join(s.ServicePath, ",")))
				}
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.COLOR_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("orange"))
				}
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.DOMAIN_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(s.ServiceDomain))
				}
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.HOSTNAME_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(s.OutboundHostname))
				}
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.PATH_TYPE_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(s.PathType))
				}
			case fmt.Sprintf("/v1/kv/%s/%s/%s", s.InstanceName, s.ServiceName, registry.SKIP_CHECK_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf("%t", s.SkipCheck)))
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer s.Server.Close()
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = getRegistrarableMock("")
	writeFeTemplateOrig := writeFeTemplate
	defer func() { writeFeTemplate = writeFeTemplateOrig }()
	writeFeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		return nil
	}
	writeBeTemplateOrig := writeBeTemplate
	defer func() { writeBeTemplate = writeBeTemplateOrig }()
	writeBeTemplate = func(filename string, data []byte, perm os.FileMode) error {
		return nil
	}
	suite.Run(t, s)
}

// Mock

type ReconfigureMock struct {
	mock.Mock
}

func (m *ReconfigureMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func (m *ReconfigureMock) GetData() (BaseReconfigure, ServiceReconfigure) {
	m.Called()
	return BaseReconfigure{}, ServiceReconfigure{}
}

func (m *ReconfigureMock) ReloadAllServices(addresses []string, instanceName, mode, listenerAddress string) error {
	params := m.Called(addresses, instanceName, mode, listenerAddress)
	return params.Error(0)
}

func (m *ReconfigureMock) GetTemplates(sr ServiceReconfigure) (front, back string, err error) {
	params := m.Called(sr)
	return params.String(0), params.String(1), params.Error(2)
}

func getReconfigureMock(skipMethod string) *ReconfigureMock {
	mockObj := new(ReconfigureMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	if skipMethod != "GetData" {
		mockObj.On("GetData", mock.Anything, mock.Anything).Return(nil)
	}
	if skipMethod != "ReloadAllServices" {
		mockObj.On("ReloadAllServices", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	if skipMethod != "GetTemplates" {
		mockObj.On("GetTemplates", mock.Anything).Return("", "", nil)
	}
	return mockObj
}

type RegistrarableMock struct {
	mock.Mock
}

func (m *RegistrarableMock) PutService(addresses []string, instanceName string, r registry.Registry) error {
	params := m.Called(addresses, instanceName, r)
	return params.Error(0)
}

func (m *RegistrarableMock) SendPutRequest(addresses []string, serviceName, key, value, instanceName string, c chan error) {
	m.Called(addresses, serviceName, key, value, instanceName, c)
}

func (m *RegistrarableMock) DeleteService(addresses []string, serviceName, instanceName string) error {
	params := m.Called(addresses, serviceName, instanceName)
	return params.Error(0)
}

func (m *RegistrarableMock) SendDeleteRequest(addresses []string, serviceName, key, value, instanceName string, c chan error) {
	m.Called(addresses, serviceName, key, value, instanceName, c)
}

func (m *RegistrarableMock) CreateConfigs(args *registry.CreateConfigsArgs) error {
	params := m.Called(args)
	return params.Error(0)
}

func (m *RegistrarableMock) GetServiceAttribute(addresses []string, instanceName, serviceName, key string) (string, error) {
	params := m.Called(addresses, instanceName, serviceName, key)
	if serviceName == "path" {
		return "path/to/my/service/api,path/to/my/other/service/api", params.Error(0)
	}
	return "something", params.Error(0)
}

func getRegistrarableMock(skipMethod string) *RegistrarableMock {
	mockObj := new(RegistrarableMock)
	if skipMethod != "PutService" {
		mockObj.On("PutService", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	if skipMethod != "SendPutRequest" {
		mockObj.On("SendPutRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
	if skipMethod != "DeleteService" {
		mockObj.On("DeleteService", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	if skipMethod != "SendDeleteRequest" {
		mockObj.On("SendDeleteRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
	if skipMethod != "CreateConfigs" {
		mockObj.On("CreateConfigs", mock.Anything).Return(nil)
	}
	if skipMethod != "GetServiceAttribute" {
		mockObj.On("GetServiceAttribute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}
	return mockObj
}

// Util

func (s ReconfigureTestSuite) verifyDoesNotPutDataToConsul(mode string) {
	s.reconfigure.Mode = mode
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	consulAddress := s.ConsulAddress
	defer func() { s.reconfigure.ConsulAddresses = []string{consulAddress} }()
	s.reconfigure.ConsulAddresses = []string{}

	s.reconfigure.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "PutService", mock.Anything, mock.Anything, mock.Anything)
}
