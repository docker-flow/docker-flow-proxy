package actions

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"

	"../proxy"
)

const serviceTemplateFeFilename = "service-formatted-fe.ctmpl"
const serviceTemplateBeFilename = "service-formatted-be.ctmpl"

// Reconfigurable defines mandatory interface
type Reconfigurable interface {
	Execute(reloadAfter bool) error
	GetData() (BaseReconfigure, proxy.Service)
	GetTemplates() (front, back string, err error)
}

// Reconfigure structure holds data required to reconfigure the proxy
type Reconfigure struct {
	BaseReconfigure
	proxy.Service
}

// BaseReconfigure contains base data required to reconfigure the proxy
type BaseReconfigure struct {
	ConfigsPath   string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	InstanceName  string `long:"proxy-instance-name" env:"PROXY_INSTANCE_NAME" default:"docker-flow" required:"true" description:"The name of the proxy instance."`
	TemplatesPath string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

// NewReconfigure creates new instance of the Reconfigurable interface
var NewReconfigure = func(baseData BaseReconfigure, serviceData proxy.Service) Reconfigurable {
	return &Reconfigure{
		BaseReconfigure: baseData,
		Service:         serviceData,
	}
}

// Execute creates a new configuration and reloads the proxy
func (m *Reconfigure) Execute(reloadAfter bool) error {
	if strings.EqualFold(os.Getenv("FILTER_PROXY_INSTANCE_NAME"), "true") &&
		!strings.EqualFold(m.InstanceName, m.Service.ProxyInstanceName) {
		logPrintf("Filtering %s configuration, with proxyInstanceName: %s", m.ServiceName, m.Service.ProxyInstanceName)
		return nil
	}
	if strings.EqualFold(os.Getenv("SKIP_ADDRESS_VALIDATION"), "false") {
		host := m.ServiceName
		if len(m.ServiceDest) > 0 && len(m.ServiceDest[0].OutboundHostname) > 0 {
			host = m.ServiceDest[0].OutboundHostname
		}
		if _, err := lookupHost(host); err != nil {
			logPrintf("Could not reach the service %s. Is the service running and connected to the same network as the proxy?", host)
			return err
		}
	}
	// Not global and replicas == 0, the service is not active
	if !m.Service.IsGlobal && m.Service.Replicas == 0 {
		action := NewRemove(
			m.Service.ServiceName,
			m.Service.AclName,
			m.ConfigsPath,
			m.TemplatesPath,
			m.InstanceName,
		)
		return action.Execute([]string{})
	}
	if err := m.createConfigsAddService(); err != nil {
		return err
	}
	if reloadAfter {
		reload := reload{}
		if err := reload.Execute(true); err != nil {
			logPrintf(err.Error())
			action := NewRemove(
				m.Service.ServiceName,
				m.Service.AclName,
				m.ConfigsPath,
				m.TemplatesPath,
				m.InstanceName,
			)
			action.Execute([]string{})
			return err
		}
	}
	return nil
}

func (m *Reconfigure) createConfigsAddService() error {
	configProxyMu.Lock()
	defer configProxyMu.Unlock()

	if err := m.createConfigs(); err != nil {
		return err
	}
	if !m.hasTemplate() {
		proxy.Instance.AddService(m.Service)
	}
	return nil
}

// GetData returns structure with reconfiguration data and the service
func (m *Reconfigure) GetData() (BaseReconfigure, proxy.Service) {
	return m.BaseReconfigure, m.Service
}

// GetTemplates returns frontend and backend templates
func (m *Reconfigure) GetTemplates() (front, back string, err error) {
	sr := &m.Service
	if value, err := strconv.ParseBool(os.Getenv("CHECK_RESOLVERS")); err == nil {
		sr.CheckResolvers = value
	}
	proxy.FormatServiceForTemplates(sr)
	if len(sr.TemplateFePath) > 0 {
		feTmpl, err := readTemplateFile(sr.TemplateFePath)
		if err != nil {
			return "", "", err
		}
		front = m.parseFrontTemplate(string(feTmpl), sr)
	}
	if len(sr.TemplateBePath) > 0 {
		beTmpl, err := readTemplateFile(sr.TemplateBePath)
		if err != nil {
			return "", "", err
		}
		back = m.parseBackTemplate(string(beTmpl), "", sr)
	} else {
		back = m.parseBackTemplate(proxy.GetBackTemplate(sr), m.getUsersList(sr), sr)
	}
	return front, back, nil
}

func (m *Reconfigure) createConfigs() error {
	templatesPath := m.TemplatesPath
	sr := &m.Service
	logPrintf("Creating configuration for the service %s", sr.ServiceName)
	feTemplate, beTemplate, err := m.GetTemplates()
	if err != nil {
		return err
	}
	if len(sr.AclName) == 0 {
		sr.AclName = sr.ServiceName
	}
	destFe := fmt.Sprintf("%s/%s-fe.cfg", templatesPath, sr.AclName)
	writeFeTemplate(destFe, []byte(feTemplate), 0664)
	destBe := fmt.Sprintf("%s/%s-be.cfg", templatesPath, sr.AclName)
	writeBeTemplate(destBe, []byte(beTemplate), 0664)
	return nil
}

func (m *Reconfigure) getUsersList(sr *proxy.Service) string {
	if len(sr.Users) > 0 {
		return `userlist {{.ServiceName}}Users{{range .Users}}
    user {{.Username}} {{if .PassEncrypted}}password{{end}}{{if not .PassEncrypted}}insecure-password{{end}} {{.Password}}{{end}}

`
	}
	return ""
}

func (m *Reconfigure) parseFrontTemplate(src string, sr *proxy.Service) string {
	var buf bytes.Buffer
	if len(src) > 0 {
		tmpl, _ := template.New("").Parse(src)
		tmpl.Execute(&buf, sr)
	}
	return buf.String()
}

func (m *Reconfigure) parseBackTemplate(src, usersList string, sr *proxy.Service) string {
	tmplUsersList, _ := template.New("template").Parse(usersList)
	tmpl, _ := template.New("").Parse(src)
	var bufUsersList bytes.Buffer
	var buf bytes.Buffer
	tmplUsersList.Execute(&bufUsersList, sr)
	tmpl.Execute(&buf, sr)
	return bufUsersList.String() + buf.String()
}

func (m *Reconfigure) hasTemplate() bool {
	return len(m.TemplateBePath) != 0 || len(m.TemplateFePath) != 0
}
