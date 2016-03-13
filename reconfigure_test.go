package main
import (
	"github.com/stretchr/testify/suite"
	"strings"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

type ReconfigureTestSuite struct {
	suite.Suite
	ServiceName		string
	ConsulAddress	string
	ConsulTemplate	string
	reconfigure		Reconfigure
}

func (s *ReconfigureTestSuite) SetupTest() {
	ConfigsDir = "test_configs"
	s.ServiceName = "myService"
	s.ConsulAddress = "http://1.2.3.4:1234"
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
	s.reconfigure = Reconfigure{
		ConsulAddress: s.ConsulAddress,
		ServiceName: s.ServiceName,
	}
//	s.reconfigure.HaProxy = getHaProxyMock()
	readFile = func(fileName string) ([]byte, error) {
		return []byte(""), nil
	}
	readDir = func (dirname string) ([]os.FileInfo, error) {
		return nil, nil
	}
	writeFile = func(fileName string, data []byte, perm os.FileMode) error {
		return nil
	}
	execHaCmd = func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{}
	}
	execConsulCmd = func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{}
	}
}

// getConsulTemplate

func (s ReconfigureTestSuite) Test_GetConsulTemplate_ReturnsFormattedContent() {
	actual := s.reconfigure.getConsulTemplate()

	s.Equal(s.ConsulTemplate, actual)
}

// Execute

func (s ReconfigureTestSuite) Test_Execute_CreatesConsulTemplate() {
	var actual string
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		if len(actual) == 0 {
			actual = string(data)
		}
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(s.ConsulTemplate, actual)
}

func (s ReconfigureTestSuite) Test_Execute_WritesToFile() {
	var actual string
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		if len(actual) == 0 {
			actual = filename
		}
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(ConsulTemplatePath, actual)
}

func (s ReconfigureTestSuite) Test_Execute_SetsFilePermissions() {
	var actual os.FileMode
	var expected os.FileMode = 0664
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actual = perm
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, actual)
}

func (s ReconfigureTestSuite) Test_Execute_RunsConsulTemplate() {
	actual := HaProxyTestSuite{}.mockConsulExecCmd()
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

	s.reconfigure.Execute([]string{})

	s.Equal(expected, *actual)
}

func (s ReconfigureTestSuite) Test_Execute_SavesConfigsToTheFile() {
	var actualFilenames []string = []string{}
	var actualData string
	writeFile = func(fileName string, data []byte, perm os.FileMode) error {
		actualFilenames = append(actualFilenames, fileName)
		actualData = string(data)
		return nil
	}

	s.reconfigure.Execute([]string{})

	s.Equal(ConfigsDir, actualFilenames[1])
}

func (s ReconfigureTestSuite) Test_Execute_ReturnsError_WhenGetConfigsFail() {
	ConfigsDir = "/this/path/does/not/exist"
	err := s.reconfigure.Execute([]string{})
	s.Error(err)
}

func (s ReconfigureTestSuite) Test_Execute_RunsHaProxy() {
	actual := HaProxyTestSuite{}.mockHaExecCmd()
	expected := []string{
		"haproxy",
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
		"-sf",
		"$(cat /var/run/haproxy.pid)",
	}

	s.reconfigure.Execute([]string{})

	s.Equal(expected, *actual)
}

// Suite

func TestReconfigureTestSuite(t *testing.T) {
	suite.Run(t, new(ReconfigureTestSuite))
}
