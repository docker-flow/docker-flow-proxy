// +build !integration

package proxy

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Setup

type HaProxyTestSuite struct {
	suite.Suite
	TemplatesPath   string
	ConfigsPath     string
	Pid             string
	TemplateContent string
	ServicesContent string
}

// Suite

func TestHaProxyUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	s := new(HaProxyTestSuite)
	s.TemplateContent = `global
    pidfile /var/run/haproxy.pid
    tune.ssl.default-dh-param 2048

defaults
    mode    http
    balance roundrobin

    option  dontlognull
    option  dontlog-normal
    option  http-server-close
    option  forwardfor
    option  redispatch

    errorfile 400 /errorfiles/400.http
    errorfile 403 /errorfiles/403.http
    errorfile 405 /errorfiles/405.http
    errorfile 408 /errorfiles/408.http
    errorfile 429 /errorfiles/429.http
    errorfile 500 /errorfiles/500.http
    errorfile 502 /errorfiles/502.http
    errorfile 503 /errorfiles/503.http
    errorfile 504 /errorfiles/504.http

    maxconn 5000
    timeout connect 5s
    timeout client  20s
    timeout server  20s
    timeout queue   30s
    timeout http-request 5s
    timeout http-keep-alive 15s

    stats enable
    stats refresh 30s
    stats realm Strictly\ Private
    stats auth admin:admin
    stats uri /admin?stats

frontend services
    bind *:80
    bind *:443
    mode http`
	s.ServicesContent = `

config1 fe content

config2 fe content

config1 be content

config2 be content`
	suite.Run(t, s)
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

// AddCertName

func (s HaProxyTestSuite) Test_AddCert_StoresCertificateName() {
	dataOrig := data
	defer func() { data = dataOrig }()
	orig := Instance
	defer func() { Instance = orig }()
	p := HaProxy{}

	p.AddCert("my-cert-3")

	s.Equal(1, len(data.Certs))
	s.Equal(map[string]bool{"my-cert-3": true}, data.Certs)
}

func (s HaProxyTestSuite) Test_AddCert_DoesNotStoreDuplicates() {
	dataOrig := data
	defer func() { data = dataOrig }()
	expected := map[string]bool{"cert-1": true, "cert-2": true, "cert-3": true}
	data.Certs = expected
	orig := Instance
	defer func() { Instance = orig }()
	p := HaProxy{}

	p.AddCert("cert-3")

	s.Equal(3, len(data.Certs))
	s.Equal(expected, data.Certs)
}

// GetCerts

func (s HaProxyTestSuite) Test_GetCerts_ReturnsAllCerts() {
	dataOrig := data
	defer func() { data = dataOrig }()
	p := HaProxy{}
	data.Certs = map[string]bool{}
	expected := map[string]string{}
	for i := 1; i <= 3; i++ {
		certName := fmt.Sprintf("my-cert-%d", i)
		data.Certs[certName] = true
		expected[certName] = fmt.Sprintf("content of the certificate /certs/%s", certName)
	}
	readFileOrig := ReadFile
	defer func() { ReadFile = readFileOrig }()
	ReadFile = func(filename string) ([]byte, error) {
		content := fmt.Sprintf("content of the certificate %s", filename)
		return []byte(content), nil
	}

	actual := p.GetCerts()

	s.EqualValues(expected, actual)
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

	err := NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

	s.Error(err)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_WritesCfgContentsIntoFile() {
	var actualData string
	expectedData := fmt.Sprintf(
		"%s%s",
		s.TemplateContent,
		s.ServicesContent,
	)
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_AddsDebug() {
	debugOrig := os.Getenv("DEBUG")
	defer func() { os.Setenv("DEBUG", debugOrig) }()
	os.Setenv("DEBUG", "true")
	var actualData string
	tmpl := strings.Replace(s.TemplateContent, "tune.ssl.default-dh-param 2048", "tune.ssl.default-dh-param 2048\n    debug", -1)
	tmpl = strings.Replace(tmpl, "    option  dontlognull\n    option  dontlog-normal\n", "", -1)
	expectedData := fmt.Sprintf(
		"%s%s",
		tmpl,
		s.ServicesContent,
	)
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_AddsCert() {
	var actualFilename string
	expectedFilename := fmt.Sprintf("%s/haproxy.cfg", s.ConfigsPath)
	var actualData string
	expectedData := fmt.Sprintf(
		"%s%s",
		strings.Replace(s.TemplateContent, "bind *:443", "bind *:443 ssl crt /certs/my-cert.pem", -1),
		s.ServicesContent,
	)
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualFilename = filename
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{"my-cert.pem": true}).CreateConfigFromTemplates()

	s.Equal(expectedFilename, actualFilename)
	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_AddsUserList() {
	var actualData string
	usersOrig := os.Getenv("USERS")
	defer func() { os.Setenv("USERS", usersOrig) }()
	os.Setenv("USERS", "my-user-1:my-password-1,my-user-2:my-password-2")
	expectedData := fmt.Sprintf(
		"%s%s",
		strings.Replace(
			s.TemplateContent,
			"frontend services",
			`userlist defaultUsers
    user my-user-1 insecure-password my-password-1
    user my-user-2 insecure-password my-password-2

frontend services`,
			-1,
		),
		s.ServicesContent,
	)
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

	s.Equal(expectedData, actualData)
}

func (s HaProxyTestSuite) Test_CreateConfigFromTemplates_ReplacesValuesWithEnvVars() {
	tests := []struct {
		envKey string
		before string
		after  string
		value  string
	}{
		{"TIMEOUT_CONNECT", "timeout connect 5s", "timeout connect 999s", "999"},
		{"TIMEOUT_CLIENT", "timeout client  20s", "timeout client  999s", "999"},
		{"TIMEOUT_SERVER", "timeout server  20s", "timeout server  999s", "999"},
		{"TIMEOUT_QUEUE", "timeout queue   30s", "timeout queue   999s", "999"},
		{"TIMEOUT_HTTP_REQUEST", "timeout http-request 5s", "timeout http-request 999s", "999"},
		{"TIMEOUT_HTTP_KEEP_ALIVE", "timeout http-keep-alive 15s", "timeout http-keep-alive 999s", "999"},
		{"STATS_USER", "stats auth admin:admin", "stats auth my-user:admin", "my-user"},
		{"STATS_PASS", "stats auth admin:admin", "stats auth admin:my-pass", "my-pass"},
	}
	for _, t := range tests {
		timeoutOrig := os.Getenv(t.envKey)
		os.Setenv(t.envKey, t.value)
		var actualFilename string
		var actualData string
		expectedData := fmt.Sprintf(
			"%s%s",
			strings.Replace(s.TemplateContent, t.before, t.after, -1),
			s.ServicesContent,
		)
		writeFile = func(filename string, data []byte, perm os.FileMode) error {
			actualFilename = filename
			actualData = string(data)
			return nil
		}

		NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

		s.Equal(expectedData, actualData)

		os.Setenv(t.envKey, timeoutOrig)
	}
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
	expectedData := fmt.Sprintf(
		"%s%s",
		s.TemplateContent,
		`

    acl url_dummy path_beg /dummy
    use_backend dummy-be if url_dummy

backend dummy-be
    server dummy 1.1.1.1:1111 check`,
	)

	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		actualData = string(data)
		return nil
	}

	NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

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

	err := NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).CreateConfigFromTemplates()

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

	actualData, _ := NewHaProxy(s.TemplatesPath, s.ConfigsPath, map[string]bool{}).ReadConfig()

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

// Mocks

func (s HaProxyTestSuite) mockHaExecCmd() *[]string {
	var actualCommand []string
	cmdRunHa = func(cmd *exec.Cmd) error {
		actualCommand = cmd.Args
		return nil
	}
	return &actualCommand
}
