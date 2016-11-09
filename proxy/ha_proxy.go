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
	Certs         []string
	TemplatesPath string
	ConfigsPath   string
}

// TODO: Change to pointer
var Instance Proxy

func NewHaProxy(templatesPath, configsPath string, certs []string) Proxy {
	return HaProxy{
		TemplatesPath: templatesPath,
		ConfigsPath:   configsPath,
		Certs:         certs,
	}
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
	type Data struct {
		CertsString string
	}
	certs := []string{}
	if len(m.Certs) > 0 {
		certs = append(certs, " ssl")
		for _, cert := range m.Certs {
			certs = append(certs, fmt.Sprintf("crt /certs/%s", cert))
		}
	}
	data := Data{
		CertsString: strings.Join(certs, " "),
	}
	tmpl.Execute(&content, data)
	return content.String(), nil
}
