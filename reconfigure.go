package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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

const (
	COLOR_KEY                   = "color"
	PATH_KEY                    = "path"
	DOMAIN_KEY                  = "domain"
	PATH_TYPE_KEY               = "pathtype"
	SKIP_CHECK_KEY              = "skipcheck"
	CONSUL_TEMPLATE_FE_PATH_KEY = "consultemplatefepath"
	CONSUL_TEMPLATE_BE_PATH_KEY = "consultemplatebepath"
)

type Reconfigure struct {
	BaseReconfigure
	ServiceReconfigure
}

type ServiceReconfigure struct {
	ServiceName          string   `short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServiceColor         string   `short:"C" long:"service-color" description:"The color of the service release in case blue-green deployment is performed (e.g. blue)."`
	ServicePath          []string `short:"p" long:"service-path" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ServiceDomain        string   `long:"service-domain" description:"The domain of the service. If specified, proxy will allow access only to requests coming from that domain (e.g. my-domain.com)."`
	ConsulTemplateFePath string   `long:"consul-template-fe-path" description:"The path to the Consul Template representing snippet of the frontend configuration. If specified, proxy template will be loaded from the specified file."`
	ConsulTemplateBePath string   `long:"consul-template-be-path" description:"The path to the Consul Template representing snippet of the backend configuration. If specified, proxy template will be loaded from the specified file."`
	PathType             string
	SkipCheck            bool
	Acl                  string
	AclCondition         string
	FullServiceName      string
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

	if path, ok := m.getServiceAttribute(address, serviceName, PATH_KEY, instanceName); ok {
		sr.ServicePath = strings.Split(path, ",")
		sr.ServiceColor, _ = m.getServiceAttribute(address, serviceName, COLOR_KEY, instanceName)
		sr.ServiceDomain, _ = m.getServiceAttribute(address, serviceName, DOMAIN_KEY, instanceName)
		sr.PathType, _ = m.getServiceAttribute(address, serviceName, PATH_TYPE_KEY, instanceName)
		skipCheck, _ := m.getServiceAttribute(address, serviceName, SKIP_CHECK_KEY, instanceName)
		sr.SkipCheck, _ = strconv.ParseBool(skipCheck)
		sr.ConsulTemplateFePath, _ = m.getServiceAttribute(address, serviceName, CONSUL_TEMPLATE_FE_PATH_KEY, instanceName)
		sr.ConsulTemplateBePath, _ = m.getServiceAttribute(address, serviceName, CONSUL_TEMPLATE_BE_PATH_KEY, instanceName)
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
	if err = m.createConfig(templatesPath, ServiceTemplateFeFilename, feTemplate, sr.ServiceName, "fe"); err != nil {
		return err
	}
	if err = m.createConfig(templatesPath, ServiceTemplateBeFilename, beTemplate, sr.ServiceName, "be"); err != nil {
		return err
	}
	return nil
}

func (m *Reconfigure) createConfig(templatesPath, file, template, serviceName, confType string) error {
	src := fmt.Sprintf("%s/%s", templatesPath, file)
	writeConsulTemplateFile(src, []byte(template), 0664)
	dest := fmt.Sprintf("%s/%s-%s", templatesPath, serviceName, confType)
	if err := m.runConsulTemplateCmd(src, dest); err != nil {
		return err
	}
	return nil
}

func (m *Reconfigure) putToConsul(address string, sr ServiceReconfigure, instanceName string) error {
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	c := make(chan error)
	go m.sendPutRequest(address, sr, COLOR_KEY, sr.ServiceColor, instanceName, c)
	go m.sendPutRequest(address, sr, PATH_KEY, strings.Join(sr.ServicePath, ","), instanceName, c)
	go m.sendPutRequest(address, sr, DOMAIN_KEY, sr.ServiceDomain, instanceName, c)
	go m.sendPutRequest(address, sr, PATH_TYPE_KEY, sr.PathType, instanceName, c)
	go m.sendPutRequest(address, sr, SKIP_CHECK_KEY, fmt.Sprintf("%t", sr.SkipCheck), instanceName, c)
	go m.sendPutRequest(address, sr, CONSUL_TEMPLATE_FE_PATH_KEY, sr.ConsulTemplateFePath, instanceName, c)
	go m.sendPutRequest(address, sr, CONSUL_TEMPLATE_BE_PATH_KEY, sr.ConsulTemplateBePath, instanceName, c)
	for i := 0; i < 6; i++ {
		err := <-c
		if err != nil {
			return fmt.Errorf("Could not send data to Consul\n%s", err.Error())
		}
	}
	return nil
}

func (m *Reconfigure) sendPutRequest(address string, sr ServiceReconfigure, key, value, instanceName string, c chan error) {
	url := fmt.Sprintf("%s/v1/kv/%s/%s/%s", address, instanceName, sr.ServiceName, key)
	client := &http.Client{}
	request, _ := http.NewRequest("PUT", url, strings.NewReader(value))
	_, err := client.Do(request)
	c <- err
}

func (m *Reconfigure) runConsulTemplateCmd(src, dest string) error {
	template := fmt.Sprintf(`%s:%s.cfg`, src, dest)
	cmdArgs := []string{
		"-consul", m.getConsulAddress(m.ConsulAddress),
		"-template", template,
		"-once",
	}
	cmd := exec.Command("consul-template", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmdRunConsul(cmd); err != nil {
		return fmt.Errorf("Command %s\n%s\n", strings.Join(cmd.Args, " "), err.Error())
	}
	return nil
}

func (m *Reconfigure) getConsulAddress(address string) string {
	a := strings.ToLower(address)
	a = strings.TrimLeft(a, "http://")
	a = strings.TrimLeft(a, "https://")
	return a
}

func (m *Reconfigure) GetConsulTemplate(sr ServiceReconfigure) (front, back string, err error) {
	if len(sr.ConsulTemplateFePath) > 0 && len(sr.ConsulTemplateBePath) > 0 {
		frontend, err := m.getConsulTemplateFromFile(sr.ConsulTemplateFePath)
		if err != nil {
			return "", "", err
		}
		backend, err := m.getConsulTemplateFromFile(sr.ConsulTemplateBePath)
		if err != nil {
			return "", "", err
		}
		// TODO: Return front as well
		return frontend, backend, err
	}
	front, backend := m.getConsulTemplateFromGo(sr)
	return front, backend, nil
}

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
    {{"{{"}}range $i, $e := service "{{.FullServiceName}}" "any"{{"}}"}}
    server {{"{{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}}"}}{{if eq .SkipCheck false}} check{{end}}
    {{"{{end}}"}}`
	tmplFront, _ := template.New("consulTemplate").Parse(srcFront)
	tmplBack, _ := template.New("consulTemplate").Parse(srcBack)
	var ctFront bytes.Buffer
	var ctBack bytes.Buffer
	tmplFront.Execute(&ctFront, sr)
	tmplBack.Execute(&ctBack, sr)
	return ctFront.String(), ctBack.String()
}

func (m *Reconfigure) getConsulTemplateFromFile(path string) (string, error) {
	content, err := readTemplateFile(path)
	if err != nil {
		return "", fmt.Errorf("Could not read the file %s\n%s", path, err.Error())
	}
	return string(content), nil
}
