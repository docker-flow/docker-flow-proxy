package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os/exec"
	"strings"
	"os"
	"fmt"
	"io/ioutil"
)

// Setup

type HaProxyTestSuite struct {
	suite.Suite
	ServiceName		string
	ConsulTemplate	string
	ConsulAddress	string
	TestConfigsDir	string
}

func (s *HaProxyTestSuite) SetupTest() {
	s.ServiceName = "myService"
	s.ConsulAddress = "http://1.2.3.4:1234"
	s.TestConfigsDir = "test_configs"
	s.ConsulTemplate = strings.TrimSpace(fmt.Sprintf(`
frontend myService-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_myService path_beg %s
	use_backend myService-be if url_myService

backend ${SERVICE_NAME}-be
	{{range service "myService" "any"}}
	server {{.Node}}_{{.Port}} {{.Address}}:{{.Port}} check
	{{end}}`, ConsulTemplatePath))
	readFile = func(fileName string) ([]byte, error) {
		return []byte(""), nil
	}
	readDir = func (dirname string) ([]os.FileInfo, error) {
		return nil, nil
	}
	writeFile = func(fileName string, data []byte, perm os.FileMode) error {
		return nil
	}
	execCmd = func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{}
	}
}

// Run

func (s HaProxyTestSuite) Test_Run_ExecutesCommand() {
	actual := s.mockExecCmd()
	expected := []string{
		"haproxy",
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	HaProxy{}.Run([]string{})

	s.Equal(expected, *actual)
}

func (s HaProxyTestSuite) Test_Run_ExecutesCommandWithExtraArgs() {
	actual := s.mockExecCmd()
	extraArgs := []string{
		"-extra-arg",
		"extra-arg-value",
	}
	expected := []string{
		"haproxy",
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	expected = append(expected, extraArgs...)
	HaProxy{}.Run(extraArgs)

	s.Equal(expected, *actual)
}



// getConsulTemplate

func (s HaProxyTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent() {
	actual := HaProxy{}.getConsulTemplate(s.ServiceName)

	s.Equal(s.ConsulTemplate, actual)
}

// CreateConfig

func (s HaProxyTestSuite) Test_CreateConfig_CreatesConsulTemplate() {
	var actual string
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		if len(actual) == 0 {
			actual = string(data)
		}
		return nil
	}

	HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, s.TestConfigsDir)

	s.Equal(s.ConsulTemplate, actual)
}

func (s HaProxyTestSuite) Test_CreateConfig_WritesToFile() {
	var actual string
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		if len(actual) == 0 {
			actual = filename
		}
		return nil
	}

	HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, s.TestConfigsDir)

	s.Equal(ConsulTemplatePath, actual)
}

func (s HaProxyTestSuite) Test_CreateConfig_SetsFilePermissions() {
	var actual os.FileMode
	var expected os.FileMode = 0664
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actual = perm
		return nil
	}

	HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, s.TestConfigsDir)

	s.Equal(expected, actual)
}

func (s HaProxyTestSuite) Test_CreateConfig_RunsConsulTemplate() {
	actual := s.mockExecCmd()
	expected := []string{
		"consul-template",
		"-consul",
		s.ConsulAddress,
		"-template",
		fmt.Sprintf(
			`"%s:%s/%s.cfg"`,
			ConsulTemplatePath,
			ConsulDir,
			s.ServiceName,
		),
		"-once",
	}

	HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, s.TestConfigsDir)

	s.Equal(expected, *actual)
}

func (s HaProxyTestSuite) Test_CreateConfig_SavesConfigsToTheFile() {
	var actualFilenames []string = []string{}
	var actualData string
	writeFile = func(fileName string, data []byte, perm os.FileMode) error {
		actualFilenames = append(actualFilenames, fileName)
		actualData = string(data)
		return nil
	}

	HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, s.TestConfigsDir)

	s.Equal(ConfigsDir, actualFilenames[1])
}

func (s HaProxyTestSuite) Test_CreateConfig_ReturnsError_WhenGetConfigsFail() {
	err := HaProxy{}.CreateConfig(s.ServiceName, s.ConsulAddress, "/this/path/does/not/exist")

	s.Error(err)
}

// getConfigs

func (s HaProxyTestSuite) Test_GetConfigs_ReturnsFileContents() {
	readFile = ioutil.ReadFile
	readDir = ioutil.ReadDir
	expected := []string{
		"template content",
		"config1 content",
		"config2 content",
	}

	actual, _ := HaProxy{}.getConfigs(s.TestConfigsDir)

	s.Equal(strings.Join(expected, "\n\n"), actual)
}

func (s HaProxyTestSuite) Test_GetConfigs_ReturnsError_WhenReadFileFails() {
	readFile = func(filename string) ([]byte, error) {
		return []byte{}, fmt.Errorf("This is an error")
	}
	_, err := HaProxy{}.getConfigs(s.TestConfigsDir)

	s.Error(err)
}

// Suite

func TestHaProxyTestSuite(t *testing.T) {
	suite.Run(t, new(HaProxyTestSuite))
}

// Helper

func (s HaProxyTestSuite) mockExecCmd() *[]string {
	var actualCommand []string
	execCmd = func(name string, arg ...string) *exec.Cmd {
		actualCommand = append([]string{name}, arg...)
		return &exec.Cmd{}
	}
	return &actualCommand
}

func (s HaProxyTestSuite) mockReadFileForConfigs() (*[]string, *string) {
	var files = []string{}
	var content = ""
	counter := 0
	readFile = func(filename string) ([]byte, error) {
		files = append(files, filename)
		content = string(string(counter))
		counter += 1
		return []byte(string(counter)), nil
	}
	return &files, &content
}
