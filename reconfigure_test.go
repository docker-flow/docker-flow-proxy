// +build !integration

package main

import (
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"strings"
	"testing"
	"net/http/httptest"
	"net/http"
	"io/ioutil"
	"strconv"
	"encoding/json"
)

type ReconfigureTestSuite struct {
	suite.Suite
	ServiceReconfigure
	ConsulAddress  string
	ConsulTemplate string
	ConfigsPath    string
	TemplatesPath  string
	reconfigure    Reconfigure
	Pid            string
	Server         *httptest.Server
	PutPathResponse string
	ConsulRequestBody ServiceReconfigure
}

func (s *ReconfigureTestSuite) SetupTest() {
	s.Pid = "123"
	s.ServicePath = []string{"path/to/my/service/api", "path/to/my/other/service/api"}
	s.ServiceDomain = "my-domain.com"
	s.ConfigsPath = "path/to/configs/dir"
	s.TemplatesPath = "test_configs/tmpl"
	s.PathType = "path_beg"
	s.ConsulTemplate = `frontend myService-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_myService path_beg path/to/my/service/api path_beg path/to/my/other/service/api
	use_backend myService-be if url_myService

backend myService-be
	{{range $i, $e := service "myService" "any"}}
	server {{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check
	{{end}}`
	cmdRunHa = func(cmd *exec.Cmd) error {
		return nil
	}
	cmdRunConsul = func(cmd *exec.Cmd) error {
		return nil
	}
	readPidFile = func(fileName string) ([]byte, error) {
		return []byte(s.Pid), nil
	}
	writeConsulTemplateFile = func(fileName string, data []byte, perm os.FileMode) error {
		return nil
	}
	s.ConsulAddress = s.Server.URL
	s.reconfigure = Reconfigure{
		BaseReconfigure: BaseReconfigure{
			ConsulAddress: s.ConsulAddress,
			TemplatesPath: s.TemplatesPath,
			ConfigsPath:   s.ConfigsPath,
		},
		ServiceReconfigure: ServiceReconfigure{
			ServiceName: s.ServiceName,
			ServicePath: s.ServicePath,
			PathType: s.PathType,
		},
	}
}

// getConsulTemplate

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent() {
	actual := s.reconfigure.getConsulTemplate(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplate, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_AddsHost() {
	s.ConsulTemplate = `frontend myService-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_myService path_beg path/to/my/service/api path_beg path/to/my/other/service/api
	acl domain_myService hdr_dom(host) -i my-domain.com
	use_backend myService-be if url_myService domain_myService

backend myService-be
	{{range $i, $e := service "myService" "any"}}
	server {{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check
	{{end}}`
	s.reconfigure.ServiceDomain = s.ServiceDomain
	actual := s.reconfigure.getConsulTemplate(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplate, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_UsesPathReg() {
	s.ConsulTemplate = strings.Replace(s.ConsulTemplate, "path_beg", "path_reg", -1)
	s.reconfigure.PathType = "path_reg"
	actual := s.reconfigure.getConsulTemplate(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplate, actual)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_AddsColor() {
	s.reconfigure.ServiceColor = "black"
	expected := fmt.Sprintf(`service "%s-%s"`, s.ServiceName, s.reconfigure.ServiceColor)

	actual := s.reconfigure.getConsulTemplate(s.reconfigure.ServiceReconfigure)

	s.Contains(actual, expected)
}

func (s ReconfigureTestSuite) Test_GetConsulTemplate_DoesNotSetCheckWhenSkipCheckIsTrue() {
	s.ConsulTemplate = strings.Replace(s.ConsulTemplate, " check", "", -1)
	s.reconfigure.SkipCheck = true
	actual := s.reconfigure.getConsulTemplate(s.reconfigure.ServiceReconfigure)

	s.Equal(s.ConsulTemplate, actual)
}

// Execute

func (s ReconfigureTestSuite) Test_Execute_CreatesConsulTemplate() {
	var actual string
	writeConsulTemplateFile = func(filename string, data []byte, perm os.FileMode) error {
		if len(actual) == 0 {
			actual = string(data)
		}
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(s.ConsulTemplate, actual)
}

func (s ReconfigureTestSuite) Test_Execute_WritesTemplateToFile() {
	var actual string
	expected := fmt.Sprintf("%s/%s", s.TemplatesPath, ServiceTemplateFilename)
	writeConsulTemplateFile = func(filename string, data []byte, perm os.FileMode) error {
		actual = filename
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_Execute_SetsFilePermissions() {
	var actual os.FileMode
	var expected os.FileMode = 0664
	writeConsulTemplateFile = func(filename string, data []byte, perm os.FileMode) error {
		actual = perm
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_Execute_RunsConsulTemplate() {
	actual := ReconfigureTestSuite{}.mockConsulExecCmd()
	expected := []string{
		"consul-template",
		"-consul",
		strings.Replace(s.ConsulAddress, "http://", "", -1),
		"-template",
		fmt.Sprintf(
			`%s/%s:%s/%s.cfg`,
			s.TemplatesPath,
			ServiceTemplateFilename,
			s.TemplatesPath,
			s.ServiceName,
		),
		"-once",
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, *actual)
}

func (s ReconfigureTestSuite) Test_Execute_RunsConsulTemplateWithTrimmedHttp() {
	actual := ReconfigureTestSuite{}.mockConsulExecCmd()
	expected := []string{
		"consul-template",
		"-consul",
		strings.Replace(s.ConsulAddress, "http://", "", -1),
		"-template",
		fmt.Sprintf(
			`%s/%s:%s/%s.cfg`,
			s.TemplatesPath,
			ServiceTemplateFilename,
			s.TemplatesPath,
			s.ServiceName,
		),
		"-once",
	}
	s.reconfigure.ConsulAddress = fmt.Sprintf("HttP://%s", s.ConsulAddress)

	s.reconfigure.Execute([]string{})

	s.Equal(expected, *actual)
}

func (s ReconfigureTestSuite) Test_Execute_RunsConsulTemplateWithTrimmedHttps() {
	actual := ReconfigureTestSuite{}.mockConsulExecCmd()
	expected := []string{
		"consul-template",
		"-consul",
		strings.Replace(s.ConsulAddress, "http://", "", -1),
		"-template",
		fmt.Sprintf(
			`%s/%s:%s/%s.cfg`,
			s.TemplatesPath,
			ServiceTemplateFilename,
			s.TemplatesPath,
			s.ServiceName,
		),
		"-once",
	}
	s.reconfigure.ConsulAddress = fmt.Sprintf("HttPs://%s", s.ConsulAddress)
	s.reconfigure.ConsulAddress = s.ConsulAddress

	s.reconfigure.Execute([]string{})

	s.Equal(expected, *actual)
}

func (s ReconfigureTestSuite) Test_Execute_ReturnsError_WhenConsulTemplateCommandFails() {
	cmdRunConsul = func(cmd *exec.Cmd) error {
		return fmt.Errorf("This is an error")
	}

	err := s.reconfigure.Execute([]string{})

	s.Error(err)
}

func (s ReconfigureTestSuite) Test_Execute_SavesConfigsToTheFile() {
	var actualFilename string
	var actualData string
	expected := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	writeFile = func(fileName string, data []byte, perm os.FileMode) error {
		actualFilename = fileName
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, actualFilename)
}

func (s ReconfigureTestSuite) Test_Execute_ReturnsError_WhenGetConfigsFail() {
	s.reconfigure.TemplatesPath = "/this/path/does/not/exist"

	err := s.reconfigure.Execute([]string{})
	s.Error(err)
}

func (s ReconfigureTestSuite) Test_Execute_InvokesHaProxyReload() {
	proxyOrig := proxy
	defer func() {
		proxy = proxyOrig
	}()
	mock := getProxyMock("")
	proxy = mock

	s.reconfigure.Execute([]string{})

	mock.AssertCalled(s.T(), "Reload")
}

func (s *ReconfigureTestSuite) Test_Execute_PutsColorToConsul() {
	s.SkipCheck = true
	s.reconfigure.SkipCheck = true
	s.reconfigure.ServiceDomain = s.ServiceDomain
	s.reconfigure.Execute([]string{})

	type data struct {key, value, expected string}

	d := []data{
		data{"color", s.ConsulRequestBody.ServiceColor, s.ServiceColor},
		data{"path", strings.Join(s.ConsulRequestBody.ServicePath, ","), strings.Join(s.ServicePath, ",")},
		data{"domain", s.ConsulRequestBody.ServiceDomain, s.ServiceDomain},
		data{"pathType", s.ConsulRequestBody.PathType, s.PathType},
		data{"skipCheck", fmt.Sprintf("%t", s.ConsulRequestBody.SkipCheck), fmt.Sprintf("%t", s.SkipCheck)},
	}
	for _, e := range d {
		s.Equal(e.expected, e.value)
	}
}

func (s *ReconfigureTestSuite) Test_Execute_ReturnsError_WhenPutToConsulFails() {
	s.reconfigure.ConsulAddress = "http:///THIS/URL/DOES/NOT/EXIST"
	actual := s.reconfigure.Execute([]string{})

	s.Error(actual)
}

func (s *ReconfigureTestSuite) Test_Execute_AddsHttpIfNotPresentInPutToConsul() {
	s.reconfigure.ConsulAddress = strings.Replace(s.ConsulAddress, "http://", "", -1)
	s.reconfigure.Execute([]string{})

	s.Equal(s.ServiceColor, s.ConsulRequestBody.ServiceColor)
}

func (s *ReconfigureTestSuite) Test_Execute_SendsServicePathToConsul() {
	s.reconfigure.Execute([]string{})

	s.Equal(s.reconfigure.ServiceColor, s.ConsulRequestBody.ServiceColor)
}

// NewReconfigure

func (s ReconfigureTestSuite) Test_NewReconfigure_AddsBaseAndService() {
	br := BaseReconfigure{ConsulAddress: "myConsulAddress"}
	sr := ServiceReconfigure{ServiceName: "myService"}

	r := NewReconfigure(br, sr)

	actualBr, actualSr := r.GetData()
	s.Equal(br, actualBr)
	s.Equal(sr, actualSr)
}

func (s ReconfigureTestSuite) Test_NewReconfigure_CreatesNewStruct() {
	r1 := NewReconfigure(
		BaseReconfigure{ConsulAddress: "myConsulAddress"},
		ServiceReconfigure{ServiceName: "myService"},
	)
	r2 := NewReconfigure(BaseReconfigure{}, ServiceReconfigure{})

	actualBr1, actualSr1 := r1.GetData()
	actualBr2, actualSr2 := r2.GetData()
	s.NotEqual(actualBr1, actualBr2)
	s.NotEqual(actualSr1, actualSr2)
}

// ReloadAllServices

func (s ReconfigureTestSuite) Test_ReloadAllServices_ReturnsError_WhenFail() {
	_, err := s.reconfigure.ReloadAllServices("this/address/does/not/exist")

	s.Error(err)
}

func (s ReconfigureTestSuite) Test_ReloadAllServices_ReturnsServiceReconfigure() {
	s.ServiceColor = "orange"
	actual, _ := s.reconfigure.ReloadAllServices(s.ConsulAddress)

	s.Len(actual, 1)
	s.Equal(s.ServiceReconfigure, actual[0])
}

func (s ReconfigureTestSuite) Test_ReloadAllServices_WriteTemplateToFile() {
	var actual string
	expected := fmt.Sprintf("%s/%s", s.TemplatesPath, ServiceTemplateFilename)
	writeConsulTemplateFile = func(filename string, data []byte, perm os.FileMode) error {
		actual = filename
		return nil
	}

	s.reconfigure.ReloadAllServices(s.ConsulAddress)

	s.Equal(expected, actual)
}

// Suite

func TestReconfigureTestSuite(t *testing.T) {
	s := new(ReconfigureTestSuite)
	s.ServiceName = "myService"
	s.PutPathResponse = "PUT_PATH_OK"
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "PUT" {
			defer r.Body.Close()
			body, _ := ioutil.ReadAll(r.Body)
			switch actualPath {
			case fmt.Sprintf("/v1/kv/docker-flow/%s/color", s.ServiceName):
				s.ConsulRequestBody.ServiceColor = string(body)
			case fmt.Sprintf("/v1/kv/docker-flow/%s/path", s.ServiceName):
				s.ConsulRequestBody.ServicePath = strings.Split(string(body), ",")
			case fmt.Sprintf("/v1/kv/docker-flow/%s/domain", s.ServiceName):
				s.ConsulRequestBody.ServiceDomain = string(body)
			case fmt.Sprintf("/v1/kv/docker-flow/%s/pathtype", s.ServiceName):
				s.ConsulRequestBody.PathType = string(body)
			case fmt.Sprintf("/v1/kv/docker-flow/%s/skipcheck", s.ServiceName):
				v, _ := strconv.ParseBool(string(body))
				s.ConsulRequestBody.SkipCheck = v
			}
		} else if r.Method == "GET" {
			switch actualPath {
			case "/v1/catalog/services":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				data := map[string][]string{"service1": []string{}, "service2": []string{}, s.ServiceName: []string{}}
				js, _ := json.Marshal(data)
				w.Write(js)
			case fmt.Sprintf("/v1/kv/docker-flow/%s/%s", s.ServiceName, PATH_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(strings.Join(s.ServicePath, ",")))
				}
			case fmt.Sprintf("/v1/kv/docker-flow/%s/%s", s.ServiceName, COLOR_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("orange"))
				}
			case fmt.Sprintf("/v1/kv/docker-flow/%s/%s", s.ServiceName, DOMAIN_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(s.ServiceDomain))
				}
			case fmt.Sprintf("/v1/kv/docker-flow/%s/%s", s.ServiceName, PATH_TYPE_KEY):
				if r.URL.RawQuery == "raw" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(s.PathType))
				}
			case fmt.Sprintf("/v1/kv/docker-flow/%s/%s", s.ServiceName, SKIP_CHECK_KEY):
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
	suite.Run(t, s)
}

// Mock

func (s ReconfigureTestSuite) mockConsulExecCmd() *[]string {
	var actualCommand []string
	cmdRunConsul = func(cmd *exec.Cmd) error {
		actualCommand = cmd.Args
		return nil
	}
	return &actualCommand
}

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

func getReconfigureMock(skipMethod string) *ReconfigureMock {
	mockObj := new(ReconfigureMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	return mockObj
}
