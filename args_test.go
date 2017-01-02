// +build !integration

package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"./actions"
	"./proxy"
	"./registry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ArgsTestSuite struct {
	suite.Suite
	args Args
}

func (s *ArgsTestSuite) SetupTest() {
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return nil
	}
	actions.OsRemove = func(name string) error {
		return nil
	}
}

// NewArgs

func (s ArgsTestSuite) Test_NewArgs_ReturnsNewStruct() {
	a := NewArgs()

	s.IsType(Args{}, a)
}

// Parse

func (s ArgsTestSuite) Test_Parse_ReturnsError_WhenFailure() {
	os.Args = []string{"myProgram", "myCommand", "--this-flag-does-not-exist=something"}

	actual := Args{}.Parse()

	s.Error(actual)
}

// Parse > Remove

func (s ArgsTestSuite) Test_Parse_ParsesRemoveLongArgsStrings() {
	os.Args = []string{"myProgram", "remove"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"serviceNameFromArgs", "service-name", &actions.RemoveInstance.ServiceName},
		{"templatesPathFromArgs", "templates-path", &actions.RemoveInstance.TemplatesPath},
		{"configsPathFromArgs", "configs-path", &actions.RemoveInstance.ConfigsPath},
		{"instanceNameFromArgs", "proxy-instance-name", &actions.RemoveInstance.InstanceName},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("--%s", d.key), d.expected)
	}
	err := Args{}.Parse()
	s.NoError(err)
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesRemoveShortArgsStrings() {
	os.Args = []string{"myProgram", "remove"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"serviceNameFromArgs", "s", &actions.RemoveInstance.ServiceName},
		{"templatesPathFromArgs", "t", &actions.RemoveInstance.TemplatesPath},
		{"configsPathFromArgs", "c", &actions.RemoveInstance.ConfigsPath},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("-%s", d.key), d.expected)
	}
	err := Args{}.Parse()
	s.NoError(err)
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_RemoveDefaultsToEnvVars() {
	os.Args = []string{"myProgram", "remove"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"proxyInstanceNameFromEnv", "PROXY_INSTANCE_NAME", &actions.RemoveInstance.InstanceName},
	}

	for _, d := range data {
		os.Setenv(d.key, d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

// Parse > Server

func (s ArgsTestSuite) Test_Parse_ParsesServerLongArgs() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromArgs", "ip", &serverImpl.IP},
		{"portFromArgs", "port", &serverImpl.Port},
		{"modeFromArgs", "mode", &serverImpl.Mode},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("--%s", d.key), d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesServerShortArgs() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromArgs", "i", &serverImpl.IP},
		{"portFromArgs", "p", &serverImpl.Port},
		{"modeFromArgs", "m", &serverImpl.Mode},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("-%s", d.key), d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ServerHasDefaultValues() {
	os.Args = []string{"myProgram", "server"}
	os.Unsetenv("IP")
	os.Unsetenv("PORT")
	data := []struct {
		expected string
		value    *string
	}{
		{"0.0.0.0", &serverImpl.IP},
		{"8080", &serverImpl.Port},
	}

	println("111")
	Args{}.Parse()
	println("999")
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ServerDefaultsToEnvVars() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromEnv", "IP", &serverImpl.IP},
		{"portFromEnv", "PORT", &serverImpl.Port},
		{"modeFromEnv", "MODE", &serverImpl.Mode},
	}

	for _, d := range data {
		os.Setenv(d.key, d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

// Suite

func TestArgsUnitTestSuite(t *testing.T) {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	logPrintf = func(format string, v ...interface{}) {}
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxy.Instance = getProxyMock("")
	suite.Run(t, new(ArgsTestSuite))
}

// Mock

//type ArgsMock struct {
//	mock.Mock
//}
//
//func (m *ArgsMock) Parse(args *Args) error {
//	params := m.Called(args)
//	return params.Error(0)
//}
//
//func getArgsMock() *ArgsMock {
//	mockObj := new(ArgsMock)
//	mockObj.On("Parse", mock.Anything).Return(nil)
//	return mockObj
//}

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

func (m *ProxyMock) AddService(service proxy.Service) {
	m.Called(service)
}

func (m *ProxyMock) RemoveService(service string) {
	m.Called(service)
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
	if skipMethod != "AddService" {
		mockObj.On("AddService", mock.Anything)
	}
	if skipMethod != "RemoveService" {
		mockObj.On("RemoveService", mock.Anything)
	}
	return mockObj
}
