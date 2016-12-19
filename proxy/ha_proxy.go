package proxy

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
)

type HaProxy struct {
	TemplatesPath string
	ConfigsPath   string
}

// TODO: Change to pointer
var Instance Proxy

type ConfigData struct {
	CertsString          string
	TimeoutConnect       string
	TimeoutClient        string
	TimeoutServer        string
	TimeoutQueue         string
	TimeoutHttpRequest   string
	TimeoutHttpKeepAlive string
	StatsUser            string
	StatsPass            string
	UserList             string
	ExtraGlobal          string
	ExtraDefaults        string
	ExtraFrontend        string
}

func NewHaProxy(templatesPath, configsPath string, certs map[string]bool) Proxy {
	data.Certs = certs
	return HaProxy{
		TemplatesPath: templatesPath,
		ConfigsPath:   configsPath,
	}
}

func (m HaProxy) AddCert(certName string) {
	if data.Certs == nil {
		data.Certs = map[string]bool{}
	}
	data.Certs[certName] = true
}

func (m HaProxy) GetCerts() map[string]string {
	certs := map[string]string{}
	for cert, _ := range data.Certs {
		content, _ := ReadFile(fmt.Sprintf("/certs/%s", cert))
		certs[cert] = string(content)
	}
	return certs
}

func (m HaProxy) RunCmd(extraArgs []string) error {
	args := []string{
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	args = append(args, extraArgs...)
	cmd := exec.Command("haproxy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmdRunHa(cmd); err != nil {
		configData, _ := readConfigsFile("/cfg/haproxy.cfg")
		return fmt.Errorf("Command %s\n%s\n%s", strings.Join(cmd.Args, " "), err.Error(), string(configData))
	}
	return nil
}

func (m HaProxy) CreateConfigFromTemplates() error {
	configsContent, err := m.getConfigs()
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%s/haproxy.cfg", m.ConfigsPath)
	return writeFile(configPath, []byte(configsContent), 0664)
}

func (m HaProxy) ReadConfig() (string, error) {
	configPath := fmt.Sprintf("%s/haproxy.cfg", m.ConfigsPath)
	out, err := ReadFile(configPath)
	if err != nil {
		return "", err
	}
	return string(out[:]), nil
}

func (m HaProxy) Reload() error {
	logPrintf("Reloading the proxy")
	pidPath := "/var/run/haproxy.pid"
	pid, err := readPidFile(pidPath)
	if err != nil {
		return fmt.Errorf("Could not read the %s file\n%s", pidPath, err.Error())
	}
	cmdArgs := []string{"-sf", string(pid)}
	return HaProxy{}.RunCmd(cmdArgs)
}

func (m HaProxy) getConfigs() (string, error) {
	contentArr := []string{}
	configsFiles := []string{"haproxy.tmpl"}
	configs, err := readConfigsDir(m.TemplatesPath)
	if err != nil {
		return "", fmt.Errorf("Could not read the directory %s\n%s", m.TemplatesPath, err.Error())
	}
	for _, fi := range configs {
		if strings.HasSuffix(fi.Name(), "-fe.cfg") {
			configsFiles = append(configsFiles, fi.Name())
		}
	}
	for _, fi := range configs {
		if strings.HasSuffix(fi.Name(), "-be.cfg") {
			configsFiles = append(configsFiles, fi.Name())
		}
	}
	for _, file := range configsFiles {
		templateBytes, err := readConfigsFile(fmt.Sprintf("%s/%s", m.TemplatesPath, file))
		if err != nil {
			return "", fmt.Errorf("Could not read the file %s\n%s", file, err.Error())
		}
		contentArr = append(contentArr, string(templateBytes))
	}
	if len(configsFiles) == 1 {
		contentArr = append(contentArr, `    acl url_dummy path_beg /dummy
    use_backend dummy-be if url_dummy

backend dummy-be
    server dummy 1.1.1.1:1111 check`)
	}
	tmpl, _ := template.New("contentTemplate").Parse(
		strings.Join(contentArr, "\n\n"),
	)
	var content bytes.Buffer
	tmpl.Execute(&content, m.getConfigData())
	return content.String(), nil
}

func (m HaProxy) getConfigData() ConfigData {
	certs := []string{}
	if len(data.Certs) > 0 {
		certs = append(certs, " ssl")
		for cert, _ := range data.Certs {
			certs = append(certs, fmt.Sprintf("crt /certs/%s", cert))
		}
	}
	d := ConfigData{
		CertsString:          strings.Join(certs, " "),
		TimeoutConnect:       "5",
		TimeoutClient:        "20",
		TimeoutServer:        "20",
		TimeoutQueue:         "30",
		TimeoutHttpRequest:   "5",
		TimeoutHttpKeepAlive: "15",
		StatsUser:            "admin",
		StatsPass:            "admin",
	}
	if len(os.Getenv("TIMEOUT_CONNECT")) > 0 {
		d.TimeoutConnect = os.Getenv("TIMEOUT_CONNECT")
	}
	if len(os.Getenv("TIMEOUT_CLIENT")) > 0 {
		d.TimeoutClient = os.Getenv("TIMEOUT_CLIENT")
	}
	if len(os.Getenv("TIMEOUT_SERVER")) > 0 {
		d.TimeoutServer = os.Getenv("TIMEOUT_SERVER")
	}
	if len(os.Getenv("TIMEOUT_QUEUE")) > 0 {
		d.TimeoutQueue = os.Getenv("TIMEOUT_QUEUE")
	}
	if len(os.Getenv("TIMEOUT_HTTP_REQUEST")) > 0 {
		d.TimeoutHttpRequest = os.Getenv("TIMEOUT_HTTP_REQUEST")
	}
	if len(os.Getenv("TIMEOUT_HTTP_KEEP_ALIVE")) > 0 {
		d.TimeoutHttpKeepAlive = os.Getenv("TIMEOUT_HTTP_KEEP_ALIVE")
	}
	if len(os.Getenv("STATS_USER")) > 0 {
		d.StatsUser = os.Getenv("STATS_USER")
	}
	if len(os.Getenv("STATS_PASS")) > 0 {
		d.StatsPass = os.Getenv("STATS_PASS")
	}
	if len(os.Getenv("USERS")) > 0 {
		d.UserList = "\nuserlist defaultUsers\n"
		users := strings.Split(os.Getenv("USERS"), ",")
		for _, user := range users {
			userPass := strings.Split(user, ":")
			d.UserList = fmt.Sprintf("%s    user %s insecure-password %s\n", d.UserList, userPass[0], userPass[1])
		}
	}
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		d.ExtraGlobal += `
    debug`
	} else {
		d.ExtraDefaults += `
    option  dontlognull
    option  dontlog-normal`
	}
	d.ExtraFrontend = os.Getenv("EXTRA_FRONTEND")
	return d
}
