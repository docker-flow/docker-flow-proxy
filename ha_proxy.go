package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates(templatesPath string, configsPath string) error
	ReadConfig(configsPath string) (string, error)
	Reload() error
}

var proxy Proxy = HaProxy{}

type HaProxy struct{}

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

func (m HaProxy) CreateConfigFromTemplates(templatesPath string, configsPath string) error {
	configsContent, err := m.getConfigs(templatesPath)
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%s/haproxy.cfg", configsPath)
	return writeFile(configPath, []byte(configsContent), 0664)
}

func (m HaProxy) ReadConfig(configsPath string) (string, error) {
	configPath := fmt.Sprintf("%s/haproxy.cfg", configsPath)
	out, err := readFile(configPath)
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

func (m HaProxy) getConfigs(templatesPath string) (string, error) {
	content := []string{}
	configsFiles := []string{"haproxy.tmpl"}
	configs, err := readConfigsDir(templatesPath)
	if err != nil {
		return "", fmt.Errorf("Could not read the directory %s\n%s", templatesPath, err.Error())
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
		templateBytes, err := readConfigsFile(fmt.Sprintf("%s/%s", templatesPath, file))
		if err != nil {
			return "", fmt.Errorf("Could not read the file %s\n%s", file, err.Error())
		}
		content = append(content, string(templateBytes))
	}
	if len(configsFiles) == 1 {
		content = append(content, `frontend dummy-fe
    bind *:80
    bind *:443
    option http-server-close
    acl url_dummy path_beg /dummy
    use_backend dummy-be if url_dummy

backend dummy-be
    server dummy 1.1.1.1:1111 check`)
	}
	return strings.Join(content, "\n\n"), nil
}
