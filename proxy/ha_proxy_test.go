// +build !integration

package proxy

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"testing"
)

// Setup

type HaProxyTestSuite struct {
	suite.Suite
	TemplatesPath string
	ConfigsPath   string
	Pid           string
}

func (s *HaProxyTestSuite) SetupTest() {
	s.Pid = "123"
	s.TemplatesPath = "test_configs/tmpl"
	s.ConfigsPath = "test_configs"
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		return nil
	}
	readPidFile = func(fileName string) ([]byte, error) {
		return []byte(s.Pid), nil
	}
}

// CreateConfigFromTemplates

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_ReturnsError_WhenReadDirFails() {
	readConfigsDirOrig := readConfigsDir
	defer func() {
		readConfigsDir = readConfigsDirOrig
	}()
	readConfigsDir = func(dirname string) ([]os.FileInfo, error) {
		return nil, fmt.Errorf("Could not read the directory")
	}

	err := NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{}).CreateConfigFromTemplates()

	s.Error(err)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_WritesCfgContentsIntoFile() {
	var actualFilename string
	expectedFilename := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	var actualData string
	expectedData := `template content

config1 content

config2 content`
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{}).CreateConfigFromTemplates()

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_AddsCert() {
	var actualFilename string
	expectedFilename := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	var actualData string
	expectedData := `template content ssl crt /certs/my-cert.pem

config1 content

config2 content`
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{"my-cert.pem"}).CreateConfigFromTemplates()

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_AddsMultipleCerts() {
	var actualFilename string
	expectedFilename := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	var actualData string
	expectedData := `template content ssl crt /certs/my-cert.pem crt /certs/my-other-cert.pem

config1 content

config2 content`
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{"my-cert.pem", "my-other-cert.pem"}).CreateConfigFromTemplates()

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_WritesMockDataIfConfigsAreNotPresent() {
	var actualData string
	readConfigsDirOrig := readConfigsDir
	defer func() {
		readConfigsDir = readConfigsDirOrig
	}()
	readConfigsDir = func(dirname string) ([]os.FileInfo, error) {
		return []os.FileInfo{}, nil
	}
	expectedData := `template content

    acl url_dummy path_beg /dummy
    use_backend dummy-be if url_dummy

backend dummy-be
    server dummy 1.1.1.1:1111 check`

	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{}).CreateConfigFromTemplates()

	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_ReturnsError_WhenReadConfigsFileFails() {
	readConfigsFileOrig := readConfigsFile
	defer func() {
		readConfigsFile = readConfigsFileOrig
	}()
	readConfigsFile = func(dirname string) ([]byte, error) {
		return nil, fmt.Errorf("Could not read the directory")
	}

	err := NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{}).CreateConfigFromTemplates()

	s.Error(err)
}

// ReadConfig

func (s *HaProxyTestSuite) Test_ReadConfig_ReturnsConfig() {
	var actualFilename string
	expectedFilename := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	expectedData := `template content

config1 content

config2 content`
	readFileOrig := ReadFile
	defer func() { ReadFile = readFileOrig }()
	ReadFile = func(filename string) ([]byte, error) {
		actualFilename = filename
		return []byte(expectedData), nil
	}

	actualData, _ := NewHaProxy(s.TemplatesPath, s.ConfigsPath, []string{}).ReadConfig()

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s *HaProxyTestSuite) Test_ReadConfig_ReturnsError_WhenReadFileFails() {
	readFileOrig := ReadFile
	defer func() { ReadFile = readFileOrig }()
	ReadFile = func(filename string) ([]byte, error) {
		return []byte{}, fmt.Errorf("This is an error")
	}

	_, actual := HaProxy{}.ReadConfig()

	s.Error(actual)
}

// Reload

func (s *HaProxyTestSuite) Test_Reload_ReadsPidFile() {
	var actual string
	readPidFile = func(fileName string) ([]byte, error) {
		actual = fileName
		return []byte("12345"), nil
	}

	HaProxy{}.Reload()

	s.Equal("/var/run/haproxy.pid", actual)
}

func (s *HaProxyTestSuite) Test_Reload_ReturnsError_WhenHaCommandFails() {
	cmdRunHa = func(cmd *exec.Cmd) error {
		return fmt.Errorf("This is an error")
	}

	err := HaProxy{}.Reload()

	s.Error(err)
}

func (s *HaProxyTestSuite) Test_Reload_ReturnsError_WhenReadPidFails() {
	readPidFile = func(fileName string) ([]byte, error) {
		return []byte(""), fmt.Errorf("This is an error")
	}

	err := HaProxy{}.Reload()

	s.Error(err)
}

func (s *HaProxyTestSuite) Test_Reload_RunsRunCmd() {
	actual := HaProxyTestSuite{}.mockHaExecCmd()
	expected := []string{
		"haproxy",
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
		"-sf",
		s.Pid,
	}

	HaProxy{}.Reload()

	s.Equal(expected, *actual)
}

// Suite

func TestHaProxyUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, new(HaProxyTestSuite))
}

func (s HaProxyTestSuite) mockHaExecCmd() *[]string {
	var actualCommand []string
	cmdRunHa = func(cmd *exec.Cmd) error {
		actualCommand = cmd.Args
		return nil
	}
	return &actualCommand
}
