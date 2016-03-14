package main

import (
	"fmt"
	"strings"
	"os"
	"os/exec"
)

type Reconfigure struct {
	ServiceName 			string	`short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServicePath 			string	`short:"p" long:"service-path" required:"true" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ConsulAddress			string	`short:"a" long:"consul-address" env:"CONSUL_ADDRESS" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
	ConfigsPath				string  `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	TemplatesPath			string  `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var reconfigure Reconfigure

func (m Reconfigure) Execute(args []string) error {
	if err := m.createConfig(); err != nil {
		return err
	}
	return m.run()
}

func (m Reconfigure) createConfig() error {
	templateContent := m.getConsulTemplate()
	templatePath := fmt.Sprintf("%s/%s", m.TemplatesPath, "service-formatted.ctmpl")
	configPath := fmt.Sprintf("%s/haproxy.cfg", m.ConfigsPath)
	writeConsulTemplateFile(templatePath, []byte(templateContent), 0664)
	if err := m.runConsulTemplateCmd(); err != nil {
		return err
	}
	configsContent, err := m.getConfigs()
	if err != nil {
		return err
	}
	return writeConsulConfigFile(configPath, []byte(configsContent), 0664)
}

func (m Reconfigure) runConsulTemplateCmd() error {
	cmdArgs := []string{
		"-consul",
		m.ConsulAddress,
		"-template",
		fmt.Sprintf(
			`%s/%s:%s/%s.cfg`,
			m.TemplatesPath,
			ServiceTemplateFilename,
			m.TemplatesPath,
			m.ServiceName,
		),
		"-once",
	}
	cmd := exec.Command("consul-template", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmdRunConsul(cmd); err != nil {
		return fmt.Errorf("Command %v\n%v\n", cmd, err)
	}
	return nil
}

func (m Reconfigure) getConsulTemplate() string {
	return strings.TrimSpace(fmt.Sprintf(`
frontend %s-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_%s path_beg %s
	use_backend %s-be if url_%s

backend %s-be
	{{range service "%s" "any"}}
	server {{.Node}}_{{.Port}} {{.Address}}:{{.Port}} check
	{{end}}`, m.ServiceName, m.ServiceName, m.ServicePath, m.ServiceName, m.ServiceName, m.ServiceName, m.ServiceName))
}

func (m Reconfigure) getConfigs() (string, error) {
	if _, err := os.Stat(m.TemplatesPath); err != nil {
		return "", fmt.Errorf("Could not find the directory %s\n%#v", m.TemplatesPath, err)
	}
	content := []string{}
	configsFiles := []string{"haproxy.tmpl"}
	configs, err := readDir(m.TemplatesPath)
	if err != nil {
		return "", fmt.Errorf("Could not read the directory %s\n%#s", m.TemplatesPath, err)
	}
	for _, fi := range configs {
		if strings.HasSuffix(fi.Name(), ".cfg") {
			configsFiles = append(configsFiles, fi.Name())
		}
	}
	for _, file := range configsFiles {
		templateBytes, err := readFile(fmt.Sprintf("%s/%s", m.TemplatesPath, file))
		if err != nil {
			return "", fmt.Errorf("Could not read the file %s\n%#s", file, err)
		}
		content = append(content, string(templateBytes))
	}
	return strings.Join(content, "\n\n"), nil
}

func (m Reconfigure) run() error {
	pidPath := "/var/run/haproxy.pid"
	pid, err := readPidFile(pidPath)
	if err != nil {
		return fmt.Errorf("Could not read the %s file\n%#v", pidPath, err)
	}
	cmdArgs := []string{"-sf", string(pid)}
	return HaProxy{}.RunCmd(cmdArgs)
}
