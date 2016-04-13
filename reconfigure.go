package main

import (
	"fmt"
	"strings"
	"os"
	"os/exec"
	"html/template"
	"bytes"
)

type Reconfigurable interface {
	Executable
	GetData() (BaseReconfigure, ServiceReconfigure)
}

type Reconfigure struct {
	BaseReconfigure
	ServiceReconfigure
}

type ServiceReconfigure struct {
	ServiceName			string		`short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServiceColor  		string		`short:"C" long:"service-color" description:"The color of the service release in case blue-green deployment is performed (e.g. blue)."`
	ServicePath   		[]string	`short:"p" long:"service-path" required:"true" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ServiceDomain 		string		`long:"service-domain" description:"The domain of the service. If specified, proxy will allow access only to requests coming from that domain (e.g. my-domain.com)."`
	PathType      		string
	Acl           		string
	AclCondition  		string
	FullServiceName		string
}

type BaseReconfigure struct {
	ConsulAddress			string	`short:"a" long:"consul-address" env:"CONSUL_ADDRESS" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
	ConfigsPath				string  `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	TemplatesPath			string  `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var reconfigure Reconfigure

var NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
	return &Reconfigure{baseData, serviceData}
}

func (m *Reconfigure) Execute(args []string) error {
	if err := m.createConfig(); err != nil {
		return err
	}
	return proxy.Reload()
}

func (m *Reconfigure) GetData() (BaseReconfigure, ServiceReconfigure) {
	return m.BaseReconfigure, m.ServiceReconfigure
}

func (m *Reconfigure) createConfig() error {
	templateContent := m.getConsulTemplate()
	templatePath := fmt.Sprintf("%s/%s", m.TemplatesPath, "service-formatted.ctmpl")
	writeConsulTemplateFile(templatePath, []byte(templateContent), 0664)
	if err := m.runConsulTemplateCmd(); err != nil {
		return err
	}
	return proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath)
}

func (m *Reconfigure) runConsulTemplateCmd() error {
	address := strings.ToLower(m.ConsulAddress)
	address = strings.TrimLeft(address, "http://")
	address = strings.TrimLeft(address, "https://")
	template := fmt.Sprintf(
		`%s/%s:%s/%s.cfg`,
		m.TemplatesPath,
		ServiceTemplateFilename,
		m.TemplatesPath,
		m.ServiceName,
	)
	cmdArgs := []string{
		"-consul", address,
		"-template", template,
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

func (m *Reconfigure) getConsulTemplate() string {
	m.Acl = ""
	m.AclCondition = ""
	if (len(m.ServiceDomain) > 0) {
		m.Acl = fmt.Sprintf(`
	acl domain_%s hdr_dom(host) -i %s`,
			m.ServiceName,
			m.ServiceDomain,
		)
		m.AclCondition = fmt.Sprintf(" domain_%s", m.ServiceName)
	}
	if (len(m.ServiceColor) > 0) {
		m.FullServiceName = fmt.Sprintf("%s-%s", m.ServiceName, m.ServiceColor)
	} else {
		m.FullServiceName = m.ServiceName
	}
	if (len(m.PathType) == 0) {
		m.PathType = "path_beg"
	}
	src := `frontend {{.ServiceName}}-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_{{.ServiceName}}{{range .ServicePath}} {{$.PathType}} {{.}}{{end}}{{.Acl}}
	use_backend {{.ServiceName}}-be if url_{{.ServiceName}}{{.AclCondition}}

backend {{.ServiceName}}-be
	{{"{{"}}range $i, $e := service "{{.FullServiceName}}" "any"{{"}}"}}
	server {{"{{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}} check"}}
	{{"{{end}}"}}`
	tmpl, _ := template.New("consulTemplate").Parse(src)
	var ct bytes.Buffer
	tmpl.Execute(&ct, m)
	return ct.String()
}
