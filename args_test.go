package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os"
	"fmt"
	"github.com/stretchr/testify/mock"
	"os/exec"
	"net/http"
)

type ArgsTestSuite struct {
	suite.Suite
	args            Args
}

func (s *ArgsTestSuite) SetupTest() {
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
}

//

func (s ArgsTestSuite) Test_Parse_ReturnsError_WhenFailure() {
	os.Args = []string{"myProgram", "myCommand", "--this-flag-does-not-exist=something"}

	actual := Args{}.Parse()

	s.Error(actual)
}

// Parse > Reconfigure

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureLongArgs() {
	argsOrig := reconfigure
	defer func() { reconfigure = argsOrig }()
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"serviceNameFromArgs", "service-name", &reconfigure.ServiceName},
		{"servicePathFromArgs", "service-path", &reconfigure.ServicePath},
		{"consulAddressFromArgs", "consul-address", &reconfigure.ConsulAddress},
		{"templatesPathFromArgs", "templates-path", &reconfigure.TemplatesPath},
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

func (s ArgsTestSuite) Test_Parse_ParsesReconfigureShortArgs() {
	argsOrig := reconfigure
	defer func() { reconfigure = argsOrig }()
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	string
		key 		string
		value		*string
	}{
		{"serviceNameFromArgs", "s", &reconfigure.ServiceName},
		{"servicePathFromArgs", "p", &reconfigure.ServicePath},
		{"consulAddressFromArgs", "a", &reconfigure.ConsulAddress},
		{"templatesPathFromArgs", "t", &reconfigure.TemplatesPath},
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

func (s ArgsTestSuite) Test_Parse_ReconfigureHasDefaultValues() {
	argsOrig := reconfigure
	defer func() { reconfigure = argsOrig }()
	os.Args = []string{"myProgram", "reconfigure"}
	data := []struct{
		expected	string
		value		*string
	}{
		{"/cfg/tmpl", &reconfigure.TemplatesPath},
		{"/cfg", &reconfigure.ConfigsPath},
	}

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

// Parse > Reconfigure

func (s ArgsTestSuite) Test_Parse_ParsesServerLongArgs() {
	argsOrig := server
	defer func() { server = argsOrig }()
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
	argsOrig := server
	defer func() { server = argsOrig }()
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
	argsOrig := server
	defer func() { server = argsOrig }()
	os.Args = []string{"myProgram", "server"}
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
	defer func() {
		os.Clearenv()
	}()

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