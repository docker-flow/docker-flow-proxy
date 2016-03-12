package main

import (
	"os"
	"fmt"
	"strings"
)

const ConsulDir = "/cfg/tmpl"
const ConsulTemplatePath = "/cfg/tmpl/service-formatted.ctmpl"
const ConfigsDir = "/cfg/tmpl"

type HaProxy struct { }

func (p HaProxy) Run(extraArgs []string) error {
	return p.runCmd(extraArgs)
}

func (p HaProxy) CreateConfig(serviceName, consulAddress, configsPath string) error {
	templateContent := p.getConsulTemplate(serviceName)
	writeFile(ConsulTemplatePath, []byte(templateContent), 0664)
	args := []string{
		"-consul",
		consulAddress,
		"-template",
		fmt.Sprintf(
			`"%s:%s/%s.cfg"`,
			ConsulTemplatePath,
			ConsulDir,
			serviceName,
		),
		"-once",
	}
	execCmd("consul-template", args...)
	configsContent, err := p.getConfigs(configsPath)
	if err != nil {
		return err
	}
	writeFile(ConfigsDir, []byte(configsContent), 0664)
	return nil
}

func (p HaProxy) getConsulTemplate(serviceName string) string {
	return strings.TrimSpace(fmt.Sprintf(`
frontend %s-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_%s path_beg %s
	use_backend %s-be if url_%s

backend ${SERVICE_NAME}-be
	{{range service "%s" "any"}}
	server {{.Node}}_{{.Port}} {{.Address}}:{{.Port}} check
	{{end}}`, serviceName, serviceName, ConsulTemplatePath, serviceName, serviceName, serviceName))
}

func (p HaProxy) getConfigs(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("Directory")
	}
	content := []string{}
	configsFiles := []string{"haproxy.tmpl"}
	configs, err := readDir(path)
	if err != nil {
		return "", fmt.Errorf("Could not read the directory %s\n%#s", path, err)
	}
	for _, fi := range configs {
		if strings.HasSuffix(fi.Name(), ".cfg") {
			configsFiles = append(configsFiles, fi.Name())
		}
	}
	for _, file := range configsFiles {
		templateBytes, err := readFile(fmt.Sprintf("%s/%s", path, file))
		if err != nil {
			return "", fmt.Errorf("Could not read the file %s\n%#s", file, err)
		}
		content = append(content, string(templateBytes))
	}
	return strings.Join(content, "\n\n"), nil
}

func (p HaProxy) runCmd(extraArgs []string) error {
	args := []string{
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	args = append(args, extraArgs...)
	cmd := execCmd("haproxy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Command %v\n%v\n", cmd, err)
	}
	return nil
}