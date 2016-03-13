package main
import (
	"fmt"
	"strings"
	"os"
)

type Reconfigure struct {
	ServiceName 		string	`short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServicePath 		string	`short:"p" long:"service-path" required:"true" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ConsulAddress		string	`short:"a" long:"consul-address" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
}

var reconfigure Reconfigure

func (m Reconfigure) Execute(args []string) error {
	if err := m.createConfig(ConfigsDir); err != nil {
		return err
	}
	cmdArgs := []string{"-sf", "$(cat /var/run/haproxy.pid)"}
	return HaProxy{}.RunCmd(cmdArgs)
}

func (m Reconfigure) createConfig(dir string) error {
	templateContent := m.getConsulTemplate()
	writeFile(ConsulTemplatePath, []byte(templateContent), 0664)
	cmdArgs := []string{
		"-consul",
		m.ConsulAddress,
		"-template",
		fmt.Sprintf(
			`"%s:%s/%s.cfg"`,
			ConsulTemplatePath,
			ConsulDir,
			m.ServiceName,
		),
		"-once",
	}
	execConsulCmd("consul-template", cmdArgs...)
	configsContent, err := m.getConfigs(dir)
	if err != nil {
		return err
	}
	return writeFile(dir, []byte(configsContent), 0664)
}

func (m Reconfigure) getConsulTemplate() string {
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
	{{end}}`, m.ServiceName, m.ServiceName, ConsulTemplatePath, m.ServiceName, m.ServiceName, m.ServiceName))
}

func (m Reconfigure) getConfigs(dir string) (string, error) {
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	content := []string{}
	configsFiles := []string{"haproxy.tmpl"}
	configs, err := readDir(dir)
	if err != nil {
		return "", fmt.Errorf("Could not read the directory %s\n%#s", dir, err)
	}
	for _, fi := range configs {
		if strings.HasSuffix(fi.Name(), ".cfg") {
			configsFiles = append(configsFiles, fi.Name())
		}
	}
	for _, file := range configsFiles {
		templateBytes, err := readFile(fmt.Sprintf("%s/%s", dir, file))
		if err != nil {
			return "", fmt.Errorf("Could not read the file %s\n%#s", file, err)
		}
		content = append(content, string(templateBytes))
	}
	return strings.Join(content, "\n\n"), nil
}
