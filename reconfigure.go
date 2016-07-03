package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"./registry"
	"html/template"
	"bytes"
)

const ServiceTemplateFeFilename = "service-formatted-fe.ctmpl"
const ServiceTemplateBeFilename = "service-formatted-be.ctmpl"

var mu = &sync.Mutex{}

type Reconfigurable interface {
	Executable
	GetData() (BaseReconfigure, ServiceReconfigure)
	ReloadAllServices(address, instanceName string) error
	GetConsulTemplate(sr ServiceReconfigure) (front, back string, err error)
}

type Reconfigure struct {
	BaseReconfigure
	ServiceReconfigure
}

type ServiceReconfigure struct {
	ServiceName          string   `short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServiceColor         string   `short:"C" long:"service-color" description:"The color of the service release in case blue-green deployment is performed (e.g. blue)."`
	ServicePath          []string `short:"p" long:"service-path" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ServicePort          string
	ServiceDomain        string   `long:"service-domain" description:"The domain of the service. If specified, proxy will allow access only to requests coming from that domain (e.g. my-domain.com)."`
	ConsulTemplateFePath string   `long:"consul-template-fe-path" description:"The path to the Consul Template representing snippet of the frontend configuration. If specified, proxy template will be loaded from the specified file."`
	ConsulTemplateBePath string   `long:"consul-template-be-path" description:"The path to the Consul Template representing snippet of the backend configuration. If specified, proxy template will be loaded from the specified file."`
	PathType             string
	Port                 string
	SkipCheck            bool
	Acl                  string
	AclCondition         string
	FullServiceName      string
	Mode 				 string
}

type BaseReconfigure struct {
	ConsulAddress string `short:"a" long:"consul-address" env:"CONSUL_ADDRESS" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
	ConfigsPath   string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	InstanceName  string `long:"proxy-instance-name" env:"PROXY_INSTANCE_NAME" default:"docker-flow" required:"true" description:"The name of the proxy instance."`
	TemplatesPath string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var reconfigure Reconfigure

var NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
	return &Reconfigure{baseData, serviceData}
}

func (m *Reconfigure) Execute(args []string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := m.createConfigs(m.TemplatesPath, m.ServiceReconfigure); err != nil {
		return err
	}
	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	if err := proxy.Reload(); err != nil {
		return err
	}
	// TODO: Extend to other registries
	return m.putToConsul(m.ConsulAddress, m.ServiceReconfigure, m.InstanceName)
}

func (m *Reconfigure) GetData() (BaseReconfigure, ServiceReconfigure) {
	return m.BaseReconfigure, m.ServiceReconfigure
}

func (m *Reconfigure) ReloadAllServices(address, instanceName string) error {
	logPrintf("Configuring existing services")
	address = strings.ToLower(address)
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	servicesUrl := fmt.Sprintf("%s/v1/catalog/services", address)
	resp, err := http.Get(servicesUrl)
	if err != nil {
		return fmt.Errorf("Could not retrieve the list of services from Consul running on %s\n%s", address, err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data map[string]interface{}
	json.Unmarshal(body, &data)
	logPrintf("\tFound %d services", len(data))

	c := make(chan ServiceReconfigure)
	for key, _ := range data {
		go m.getService(address, key, instanceName, c)
	}
	for i := 0; i < len(data); i++ {
		s := <-c
		if len(s.ServicePath) > 0 {
			logPrintf("\tConfiguring %s", s.ServiceName)
			m.createConfigs(m.TemplatesPath, s)
		}
	}

	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	return proxy.Reload()
}

func (m *Reconfigure) getService(address, serviceName, instanceName string, c chan ServiceReconfigure) {
	sr := ServiceReconfigure{ServiceName: serviceName}

	if path, ok := m.getServiceAttribute(address, serviceName, registry.PATH_KEY, instanceName); ok {
		sr.ServicePath = strings.Split(path, ",")
		sr.ServiceColor, _ = m.getServiceAttribute(address, serviceName, registry.COLOR_KEY, instanceName)
		sr.ServiceDomain, _ = m.getServiceAttribute(address, serviceName, registry.DOMAIN_KEY, instanceName)
		sr.PathType, _ = m.getServiceAttribute(address, serviceName, registry.PATH_TYPE_KEY, instanceName)
		skipCheck, _ := m.getServiceAttribute(address, serviceName, registry.SKIP_CHECK_KEY, instanceName)
		sr.SkipCheck, _ = strconv.ParseBool(skipCheck)
		sr.ConsulTemplateFePath, _ = m.getServiceAttribute(address, serviceName, registry.CONSUL_TEMPLATE_FE_PATH_KEY, instanceName)
		sr.ConsulTemplateBePath, _ = m.getServiceAttribute(address, serviceName, registry.CONSUL_TEMPLATE_BE_PATH_KEY, instanceName)
	}
	c <- sr
}

func (m *Reconfigure) getServiceAttribute(address, serviceName, key, instanceName string) (string, bool) {
	url := fmt.Sprintf("%s/v1/kv/%s/%s/%s?raw", address, instanceName, serviceName, key)
	resp, _ := http.Get(url)
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), true
}

func (m *Reconfigure) createConfigs(templatesPath string, sr ServiceReconfigure) error {
	logPrintf("Creating configuration for the service %s", sr.ServiceName)
	feTemplate, beTemplate, err := m.GetConsulTemplate(sr)
	if err != nil {
		return err
	}
	args := registry.CreateConfigsArgs{
		Address: m.ConsulAddress,
		TemplatesPath: templatesPath,
		FeFile: ServiceTemplateFeFilename,
		FeTemplate: feTemplate,
		BeFile: ServiceTemplateBeFilename,
		BeTemplate: beTemplate,
		ServiceName: sr.ServiceName,
		Monitor: false,
	}
	if err = registryInstance.CreateConfigs(args); err != nil {
		return err
	}
	return nil
}

func (m *Reconfigure) putToConsul(address string, sr ServiceReconfigure, instanceName string) error {
	r := registry.Registry{
		ServiceName: sr.ServiceName,
		ServiceColor: sr.ServiceColor,
		ServicePath: sr.ServicePath,
		ServiceDomain: sr.ServiceDomain,
		PathType: sr.PathType,
		SkipCheck: sr.SkipCheck,
		ConsulTemplateFePath: sr.ConsulTemplateFePath,
		ConsulTemplateBePath: sr.ConsulTemplateBePath,
	}
	if err := registryInstance.PutService(address, instanceName, r); err != nil {
		return err
	}
	return nil
}

// TODO: Move to registry package
func (m *Reconfigure) GetConsulTemplate(sr ServiceReconfigure) (front, back string, err error) {
	if len(sr.ConsulTemplateFePath) > 0 && len(sr.ConsulTemplateBePath) > 0 {
		front, err = m.getConsulTemplateFromFile(sr.ConsulTemplateFePath)
		if err != nil {
			return "", "", err
		}
		back, err = m.getConsulTemplateFromFile(sr.ConsulTemplateBePath)
		if err != nil {
			return "", "", err
		}
	} else {
		front, back = m.getConsulTemplateFromGo(sr)
	}
	return front, back, nil
}

// TODO: Move to registry package
func (m *Reconfigure) getConsulTemplateFromGo(sr ServiceReconfigure) (frontend, backend string) {
	sr.Acl = ""
	sr.AclCondition = ""
	if len(sr.ServiceDomain) > 0 {
		sr.Acl = fmt.Sprintf(`
    acl domain_%s hdr_dom(host) -i %s`,
			sr.ServiceName,
			sr.ServiceDomain,
		)
		sr.AclCondition = fmt.Sprintf(" domain_%s", sr.ServiceName)
	}
	if len(sr.ServiceColor) > 0 {
		sr.FullServiceName = fmt.Sprintf("%s-%s", sr.ServiceName, sr.ServiceColor)
	} else {
		sr.FullServiceName = sr.ServiceName
	}
	if len(sr.PathType) == 0 {
		sr.PathType = "path_beg"
	}
	srcFront := `
    acl url_{{.ServiceName}}{{range .ServicePath}} {{$.PathType}} {{.}}{{end}}{{.Acl}}
    use_backend {{.ServiceName}}-be if url_{{.ServiceName}}{{.AclCondition}}`
	srcBack := `backend {{.ServiceName}}-be
    `
	if strings.ToLower(sr.Mode) == "service" {
		srcBack += `{{"{{"}}range $i, $e := nodes{{"}}"}}
    server {{"{{$e.Node}}_{{$i}} {{$e.Address}}:{{.Port}}"}}{{if eq .SkipCheck false}} check{{end}}`
	} else {
		srcBack += `{{"{{"}}range $i, $e := service "{{.FullServiceName}}" "any"{{"}}"}}
    server {{"{{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}}"}}{{if eq .SkipCheck false}} check{{end}}`
	}
	srcBack += `
    {{"{{end}}"}}`
	tmplFront, _ := template.New("consulTemplate").Parse(srcFront)
	tmplBack, _ := template.New("consulTemplate").Parse(srcBack)
	var ctFront bytes.Buffer
	var ctBack bytes.Buffer
	tmplFront.Execute(&ctFront, sr)
	tmplBack.Execute(&ctBack, sr)
	return ctFront.String(), ctBack.String()
}

// TODO: Move to registry package
func (m *Reconfigure) getConsulTemplateFromFile(path string) (string, error) {
	content, err := readTemplateFile(path)
	if err != nil {
		return "", fmt.Errorf("Could not read the file %s\n%s", path, err.Error())
	}
	return string(content), nil
}
