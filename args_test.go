package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os"
	"github.com/stretchr/testify/mock"
	"os/exec"
	"net/http"
	"fmt"
)

type ArgsTestSuite struct {
	suite.Suite
	args            Args
	TemplatesPath 	string
}

func (s *ArgsTestSuite) SetupTest() {
	s.TemplatesPath = "test_configs/tmpl"
	cmdRunConsul = func(cmd *exec.Cmd) error {
		return nil
	}
	cmdRunHa = func(cmd *exec.Cmd) error {
		return nil
	}
	readFile = func(fileName string) ([]byte, error) {
		return []byte(""), nil
	}
	readDir = func (dirname string) ([]os.FileInfo, error) {
		return nil, nil
	}
	writeConsulTemplateFile = func(fileName string, data []byte, perm os.FileMode) error {
		return nil
	}
	writeConsulConfigFile = func(fileName string, data []byte, perm os.FileMode) error {
		return nil
	}
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return nil
	}
	readPidFile = func(fileName string) ([]byte, error) {
		return []byte(""), nil
	}
	os.Setenv("CONSUL_ADDRESS", "myConsulAddress")
	logPrintf = func(format string, v ...interface{}) {}
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

// Parse > Reconfigure

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureLongArgsStrings() {
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"serviceNameFromArgs", "service-name", &reconfigure.ServiceName},
		{"serviceColorFromArgs", "service-color", &reconfigure.ServiceColor},
		{"serviceDomainFromArgs", "service-domain", &reconfigure.ServiceDomain},
		{"consulAddressFromArgs", "consul-address", &reconfigure.ConsulAddress},
		{s.TemplatesPath, "templates-path", &reconfigure.TemplatesPath},
		{"configsPathFromArgs", "configs-path", &reconfigure.ConfigsPath},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("--%s", d.key), d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureLongArgsSlices() {
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	[]string
		key 		string
		value		*[]string
	}{
		{[]string{"path1", "path2"}, "service-path", &reconfigure.ServicePath},
	}

	for _, d := range data {
		for _, v := range d.expected {
			os.Args = append(os.Args, fmt.Sprintf("--%s", d.key), v)
		}

	}

	Args{}.Parse()

	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureShortArgsStrings() {
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"serviceNameFromArgs", "s", &reconfigure.ServiceName},
		{"serviceColorFromArgs", "C", &reconfigure.ServiceColor},
		{"consulAddressFromArgs", "a", &reconfigure.ConsulAddress},
		{s.TemplatesPath, "t", &reconfigure.TemplatesPath},
		{"configsPathFromArgs", "c", &reconfigure.ConfigsPath},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("-%s", d.key), d.expected)
	}
	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureShortArgsSlices() {
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	[]string
		key 		string
		value		*[]string
	}{
		{[]string{"p1", "p2"}, "p", &reconfigure.ServicePath},
	}
	for _, d := range data {
		for _, v := range d.expected {
			os.Args = append(os.Args, fmt.Sprintf("-%s", d.key), v)
		}
	}

	Args{}.Parse()

	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ReconfigureHasDefaultValues() {
	os.Args = []string{
		"myProgram", "reconfigure",
		"--service-name", "myService",
		"--service-path", "my/service/path",
	}
	data := []struct{
		expected	string
		value		*string
	}{
		{"/cfg/tmpl", &reconfigure.TemplatesPath},
		{"/cfg", &reconfigure.ConfigsPath},
	}
	reconfigure.ConsulAddress = "myConsulAddress"
	reconfigure.ServicePath = []string{"p1", "p2"}
	reconfigure.ServiceName = "myServiceName"

	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ReconfigureDefaultsToEnvVars() {
	os.Args = []string{
		"myProgram", "reconfigure",
		"--service-name", "serviceName",
		"--service-path", "servicePath",
	}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"consulAddressFromEnv", "CONSUL_ADDRESS", &reconfigure.ConsulAddress},
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
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"ipFromArgs", "ip", &server.IP},
		{"portFromArgs", "port", &server.Port},
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
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"ipFromArgs", "i", &server.IP},
		{"portFromArgs", "p", &server.Port},
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
	data := []struct{
		expected	string
		value		*string
	}{
		{"0.0.0.0", &server.IP},
		{"8080", &server.Port},
	}

	Args{}.Parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
}

func (s ArgsTestSuite) Test_Parse_ServerDefaultsToEnvVars() {
	os.Args = []string{"myProgram", "server"}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"ipFromEnv", "IP", &server.IP},
		{"portFromEnv", "PORT", &server.Port},
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

func TestArgsTestSuite(t *testing.T) {
	suite.Run(t, new(ArgsTestSuite))
}

// Mock

type ArgsMock struct{
	mock.Mock
}

func (m *ArgsMock) Parse(args *Args) error {
	params := m.Called(args)
	return params.Error(0)
}

func getArgsMock() *ArgsMock {
	mockObj := new(ArgsMock)
	mockObj.On("Parse", mock.Anything).Return(nil)
	return mockObj
}