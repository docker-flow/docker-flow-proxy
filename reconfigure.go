package main

import (
	"./registry"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
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
	ReloadAllServices(addresses []string, instanceName, mode, listenerAddress string) error
	GetTemplates(sr ServiceReconfigure) (front, back string, err error)
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
	ServiceDomain        string `long:"service-domain" description:"The domain of the service. If specified, proxy will allow access only to requests coming from that domain (e.g. my-domain.com)."`
	OutboundHostname     string `long:"outbound-hostname" description:"The hostname running the service. If specified, proxy will redirect traffic to this hostname instead of using the service's name."`
	ConsulTemplateFePath string `long:"consul-template-fe-path" description:"The path to the Consul Template representing snippet of the frontend configuration. If specified, proxy template will be loaded from the specified file."`
	ConsulTemplateBePath string `long:"consul-template-be-path" description:"The path to the Consul Template representing snippet of the backend configuration. If specified, proxy template will be loaded from the specified file."`
	Mode                 string `short:"m" long:"mode" env:"MODE" description:"If set to 'swarm', proxy will operate assuming that Docker service from v1.12+ is used."`
	PathType             string
	Port                 string
	SkipCheck            bool
	Acl                  string
	AclCondition         string
	FullServiceName      string
	Host                 string
	Distribute           bool
	LookupRetry          int
	LookupRetryInterval  int
}

type BaseReconfigure struct {
	ConsulAddresses       []string
	ConfigsPath           string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	InstanceName          string `long:"proxy-instance-name" env:"PROXY_INSTANCE_NAME" default:"docker-flow" required:"true" description:"The name of the proxy instance."`
	TemplatesPath         string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
	skipAddressValidation bool
}

var reconfigure Reconfigure

var NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
	return &Reconfigure{baseData, serviceData}
}

func (m *Reconfigure) Execute(args []string) error {
	mu.Lock()
	defer mu.Unlock()
	if isSwarm(m.ServiceReconfigure.Mode) && !m.skipAddressValidation {
		host := m.ServiceName
		if len(m.OutboundHostname) > 0 {
			host = m.OutboundHostname
		}
		if _, err := lookupHost(host); err != nil {
			logPrintf("Could not reach the service %s. Is the service running and connected to the same network as the proxy?", host)
			return err
		}
	}
	if err := m.createConfigs(m.TemplatesPath, &m.ServiceReconfigure); err != nil {
		return err
	}
	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	if err := proxy.Reload(); err != nil {
		return err
	}
	if len(m.ConsulAddresses) > 0 || !isSwarm(m.ServiceReconfigure.Mode) {
		if err := m.putToConsul(m.ConsulAddresses, m.ServiceReconfigure, m.InstanceName); err != nil {
			return err
		}
	}
	return nil
}

func (m *Reconfigure) GetData() (BaseReconfigure, ServiceReconfigure) {
	return m.BaseReconfigure, m.ServiceReconfigure
}

func (m *Reconfigure) ReloadAllServices(addresses []string, instanceName, mode, listenerAddress string) error {
	if len(listenerAddress) > 0 {
		fullAddress := fmt.Sprintf("%s/v1/docker-flow-swarm-listener/notify-services", listenerAddress)
		resp, err := httpGet(fullAddress)
		if err != nil {
			return err
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Swarm Listener responded with the status code %d", resp.StatusCode)
		}
		logPrintf("A request was sent to the Swarm listener running on %s. The proxy will be reconfigured soon.", listenerAddress)
	} else if len(addresses) > 0 || !isSwarm(mode) {
		return m.reloadFromRegistry(addresses, instanceName, mode)
	}
	return nil
}

func (m *Reconfigure) reloadFromRegistry(addresses []string, instanceName, mode string) error {
	var resp *http.Response
	var err error
	logPrintf("Configuring existing services")
	found := false
	for _, address := range addresses {
		var servicesUrl string
		address = strings.ToLower(address)
		if !strings.HasPrefix(address, "http") {
			address = fmt.Sprintf("http://%s", address)
		}
		if isSwarm(mode) {
			// TODO: Test
			servicesUrl = fmt.Sprintf("%s/v1/kv/docker-flow/service?recurse", address)
		} else {
			servicesUrl = fmt.Sprintf("%s/v1/catalog/services", address)
		}
		resp, err = http.Get(servicesUrl)
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Could not retrieve the list of services from Consul")
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	c := make(chan ServiceReconfigure)
	count := 0
	if isSwarm(mode) {
		// TODO: Test
		type Key struct {
			Value string `json:"Key"`
		}
		data := []Key{}
		json.Unmarshal(body, &data)
		count = len(data)
		for _, key := range data {
			parts := strings.Split(key.Value, "/")
			serviceName := parts[len(parts)-1]
			go m.getService(addresses, serviceName, instanceName, c)
		}
	} else {
		var data map[string]interface{}
		json.Unmarshal(body, &data)
		count = len(data)
		for key, _ := range data {
			go m.getService(addresses, key, instanceName, c)
		}
	}
	logPrintf("\tFound %d services", count)

	for i := 0; i < count; i++ {
		s := <-c
		s.Mode = mode
		if len(s.ServicePath) > 0 {
			logPrintf("\tConfiguring %s", s.ServiceName)
			m.createConfigs(m.TemplatesPath, &s)
		}
	}

	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	if count == 0 && len(body) > 0 {
		return fmt.Errorf("Config response was non-empty and invalid")
	}
	return proxy.Reload()
}

func (m *Reconfigure) getService(addresses []string, serviceName, instanceName string, c chan ServiceReconfigure) {
	sr := ServiceReconfigure{ServiceName: serviceName}

	path, err := registryInstance.GetServiceAttribute(addresses, serviceName, registry.PATH_KEY, instanceName)
	if err == nil {
		sr.ServicePath = strings.Split(path, ",")
		sr.ServiceColor, _ = m.getServiceAttribute(addresses, serviceName, registry.COLOR_KEY, instanceName)
		sr.ServiceDomain, _ = m.getServiceAttribute(addresses, serviceName, registry.DOMAIN_KEY, instanceName)
		sr.OutboundHostname, _ = m.getServiceAttribute(addresses, serviceName, registry.HOSTNAME_KEY, instanceName)
		sr.PathType, _ = m.getServiceAttribute(addresses, serviceName, registry.PATH_TYPE_KEY, instanceName)
		skipCheck, _ := m.getServiceAttribute(addresses, serviceName, registry.SKIP_CHECK_KEY, instanceName)
		sr.SkipCheck, _ = strconv.ParseBool(skipCheck)
		sr.ConsulTemplateFePath, _ = m.getServiceAttribute(addresses, serviceName, registry.CONSUL_TEMPLATE_FE_PATH_KEY, instanceName)
		sr.ConsulTemplateBePath, _ = m.getServiceAttribute(addresses, serviceName, registry.CONSUL_TEMPLATE_BE_PATH_KEY, instanceName)
		sr.Port, _ = m.getServiceAttribute(addresses, serviceName, registry.PORT, instanceName)
	}
	c <- sr
}

// TODO: Remove in favour of registry.GetServiceAttribute
func (m *Reconfigure) getServiceAttribute(addresses []string, serviceName, key, instanceName string) (string, bool) {
	for _, address := range addresses {
		url := fmt.Sprintf("%s/v1/kv/%s/%s/%s?raw", address, instanceName, serviceName, key)
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			return string(body), true
		}
	}
	return "", false
}

func (m *Reconfigure) createConfigs(templatesPath string, sr *ServiceReconfigure) error {
	logPrintf("Creating configuration for the service %s", sr.ServiceName)
	feTemplate, beTemplate, err := m.GetTemplates(*sr)
	if err != nil {
		return err
	}
	if strings.EqualFold(sr.Mode, "service") || strings.EqualFold(sr.Mode, "swarm") {
		destFe := fmt.Sprintf("%s/%s-fe.cfg", templatesPath, sr.ServiceName)
		writeFeTemplate(destFe, []byte(feTemplate), 0664)
		destBe := fmt.Sprintf("%s/%s-be.cfg", templatesPath, sr.ServiceName)
		writeBeTemplate(destBe, []byte(beTemplate), 0664)
	} else {
		args := registry.CreateConfigsArgs{
			Addresses:     m.ConsulAddresses,
			TemplatesPath: templatesPath,
			FeFile:        ServiceTemplateFeFilename,
			FeTemplate:    feTemplate,
			BeFile:        ServiceTemplateBeFilename,
			BeTemplate:    beTemplate,
			ServiceName:   sr.ServiceName,
		}
		if err = registryInstance.CreateConfigs(&args); err != nil {
			return err
		}
	}
	return nil
}

func (m *Reconfigure) putToConsul(addresses []string, sr ServiceReconfigure, instanceName string) error {
	r := registry.Registry{
		ServiceName:          sr.ServiceName,
		ServiceColor:         sr.ServiceColor,
		ServicePath:          sr.ServicePath,
		ServiceDomain:        sr.ServiceDomain,
		OutboundHostname:     sr.OutboundHostname,
		PathType:             sr.PathType,
		SkipCheck:            sr.SkipCheck,
		ConsulTemplateFePath: sr.ConsulTemplateFePath,
		ConsulTemplateBePath: sr.ConsulTemplateBePath,
		Port:                 sr.Port,
	}
	if err := registryInstance.PutService(addresses, instanceName, r); err != nil {
		return err
	}
	return nil
}

// TODO: Move to registry package
func (m *Reconfigure) GetTemplates(sr ServiceReconfigure) (front, back string, err error) {
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
	sr.Host = m.ServiceName
	if len(m.OutboundHostname) > 0 {
		sr.Host = m.OutboundHostname
	}
	fmt.Println("Configuring host:", sr.Host)
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
	if strings.EqualFold(sr.Mode, "service") || strings.EqualFold(sr.Mode, "swarm") {
		srcBack += `server {{.ServiceName}} {{.Host}}:{{.Port}}`
	} else {
		srcBack += `{{"{{"}}range $i, $e := service "{{.FullServiceName}}" "any"{{"}}"}}
    server {{"{{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}}"}}{{if eq .SkipCheck false}} check{{end}}
    {{"{{end}}"}}`
	}
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
