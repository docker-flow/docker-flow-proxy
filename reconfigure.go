package main

//curl http://192.168.99.100:8500/v1/kv/books-ms?recurse | jq '.'

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

var mu = &sync.Mutex{}

type Reconfigurable interface {
	Executable
	GetData() (BaseReconfigure, ServiceReconfigure)
	ReloadAllServices(address string) error
}

const (
	COLOR_KEY      = "color"
	PATH_KEY       = "path"
	DOMAIN_KEY     = "domain"
	PATH_TYPE_KEY  = "pathtype"
	SKIP_CHECK_KEY = "skipcheck"
)

type Reconfigure struct {
	BaseReconfigure
	ServiceReconfigure
}

type ServiceReconfigure struct {
	ServiceName     string   `short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServiceColor    string   `short:"C" long:"service-color" description:"The color of the service release in case blue-green deployment is performed (e.g. blue)."`
	ServicePath     []string `short:"p" long:"service-path" required:"true" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ServiceDomain   string   `long:"service-domain" description:"The domain of the service. If specified, proxy will allow access only to requests coming from that domain (e.g. my-domain.com)."`
	PathType        string
	SkipCheck       bool
	Acl             string
	AclCondition    string
	FullServiceName string
}

type BaseReconfigure struct {
	ConsulAddress string `short:"a" long:"consul-address" env:"CONSUL_ADDRESS" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
	ConfigsPath   string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	TemplatesPath string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var reconfigure Reconfigure

var NewReconfigure = func(baseData BaseReconfigure, serviceData ServiceReconfigure) Reconfigurable {
	return &Reconfigure{baseData, serviceData}
}

func (m *Reconfigure) Execute(args []string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := m.createConfig(m.TemplatesPath, m.ServiceReconfigure); err != nil {
		return err
	}
	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	if err := proxy.Reload(); err != nil {
		return err
	}
	return m.putToConsul(m.ConsulAddress, m.ServiceReconfigure)
}

func (m *Reconfigure) GetData() (BaseReconfigure, ServiceReconfigure) {
	return m.BaseReconfigure, m.ServiceReconfigure
}

func (m *Reconfigure) ReloadAllServices(address string) error {
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
	err = json.Unmarshal(body, &data)
	logPrintf("\tFound %d services", len(data))

	c := make(chan ServiceReconfigure)
	for key, _ := range data {
		go m.getService(address, key, c)
	}
	for i := 0; i < len(data); i++ {
		s := <-c
		if len(s.ServicePath) > 0 {
			logPrintf("\tConfiguring %s", s.ServiceName)
			m.createConfig(m.TemplatesPath, s)
		}
	}

	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	return proxy.Reload()
}

func (m *Reconfigure) getService(address, serviceName string, c chan ServiceReconfigure) {
	sr := ServiceReconfigure{ServiceName: serviceName}

	if path, ok := m.getServiceAttribute(address, serviceName, PATH_KEY); ok {
		sr.ServicePath = strings.Split(path, ",")
		sr.ServiceColor, _ = m.getServiceAttribute(address, serviceName, COLOR_KEY)
		sr.ServiceDomain, _ = m.getServiceAttribute(address, serviceName, DOMAIN_KEY)
		sr.PathType, _ = m.getServiceAttribute(address, serviceName, PATH_TYPE_KEY)
		skipCheck, _ := m.getServiceAttribute(address, serviceName, SKIP_CHECK_KEY)
		sr.SkipCheck, _ = strconv.ParseBool(skipCheck)
	}
	c <- sr
}

func (m *Reconfigure) getServiceAttribute(address, serviceName, key string) (string, bool) {
	url := fmt.Sprintf("%s/v1/kv/docker-flow/%s/%s?raw", address, serviceName, key)
	resp, _ := http.Get(url)
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), true
}

func (m *Reconfigure) createConfig(templatesPath string, sr ServiceReconfigure) error {
	logPrintf("Creating configuration for the service %s", sr.ServiceName)
	templateContent := m.getConsulTemplate(sr)
	path := fmt.Sprintf("%s/%s", templatesPath, "service-formatted.ctmpl")
	writeConsulTemplateFile(path, []byte(templateContent), 0664)
	if err := m.runConsulTemplateCmd(templatesPath, sr.ServiceName); err != nil {
		return err
	}
	return nil
}

// TODO: Integration tests
func (m *Reconfigure) putToConsul(address string, sr ServiceReconfigure) error {
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	c := make(chan error)
	go m.sendPutRequest(address, sr, COLOR_KEY, sr.ServiceColor, c)
	go m.sendPutRequest(address, sr, PATH_KEY, strings.Join(sr.ServicePath, ","), c)
	go m.sendPutRequest(address, sr, DOMAIN_KEY, sr.ServiceDomain, c)
	go m.sendPutRequest(address, sr, PATH_TYPE_KEY, sr.PathType, c)
	go m.sendPutRequest(address, sr, SKIP_CHECK_KEY, fmt.Sprintf("%t", sr.SkipCheck), c)
	for i := 0; i < 5; i++ {
		err := <-c
		if err != nil {
			return fmt.Errorf("Could not send data to Consul\n%s", err.Error())
		}
	}
	return nil
}

func (m *Reconfigure) sendPutRequest(address string, sr ServiceReconfigure, key string, value string, c chan error) {
	url := fmt.Sprintf("%s/v1/kv/docker-flow/%s/%s", address, sr.ServiceName, key)
	client := &http.Client{}
	request, _ := http.NewRequest("PUT", url, strings.NewReader(value))
	_, err := client.Do(request)
	c <- err
}

func (m *Reconfigure) runConsulTemplateCmd(templatesPath, serviceName string) error {
	template := fmt.Sprintf(
		`%s/%s:%s/%s.cfg`,
		templatesPath,
		ServiceTemplateFilename,
		templatesPath,
		serviceName,
	)
	cmdArgs := []string{
		"-consul", m.getConsulAddress(m.ConsulAddress),
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

func (m *Reconfigure) getConsulAddress(address string) string {
	a := strings.ToLower(address)
	a = strings.TrimLeft(a, "http://")
	a = strings.TrimLeft(a, "https://")
	return a
}

func (m *Reconfigure) getConsulTemplate(sr ServiceReconfigure) string {
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
	src := `frontend {{.ServiceName}}-fe
	bind *:80
	bind *:443
	option http-server-close
	acl url_{{.ServiceName}}{{range .ServicePath}} {{$.PathType}} {{.}}{{end}}{{.Acl}}
	use_backend {{.ServiceName}}-be if url_{{.ServiceName}}{{.AclCondition}}

backend {{.ServiceName}}-be
	{{"{{"}}range $i, $e := service "{{.FullServiceName}}" "any"{{"}}"}}
	server {{"{{$e.Node}}_{{$i}}_{{$e.Port}} {{$e.Address}}:{{$e.Port}}"}}{{if eq .SkipCheck false}} check{{end}}
	{{"{{end}}"}}`
	tmpl, _ := template.New("consulTemplate").Parse(src)
	var ct bytes.Buffer
	tmpl.Execute(&ct, sr)
	return ct.String()
}
