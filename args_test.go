package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os"
	"fmt"
	"github.com/stretchr/testify/mock"
	"os/exec"
)

type ArgsTestSuite struct {
	suite.Suite
	args            Args
	argsReconfigure Reconfigure
}

func (s *ArgsTestSuite) SetupTest() {
	s.argsReconfigure.ServiceName = "myService"
	s.argsReconfigure.ServicePath = "/path/to/my/service"
	s.argsReconfigure.ConsulAddress = "http://1.2.3.4:1234"
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
}

// Parse

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

func (s ArgsTestSuite) Test_Parse_HasDefaultValues() {
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

func (s ArgsTestSuite) Test_Parse_DefaultsToEnvVars() {
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

func (s ArgsTestSuite) TestParseArgs_ReturnsError_WhenFailure() {
	os.Args = []string{"myProgram", "myCommand", "--this-flag-does-not-exist=something"}

	actual := Args{}.Parse()

	s.Error(actual)
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