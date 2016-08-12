package registry
import (
	"fmt"
	"net/http"
	"strings"
	"os/exec"
	"os"
	"io/ioutil"
)

type Consul struct {}

var cmdRunConsulTemplate = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var WriteConsulTemplateFile = ioutil.WriteFile

type CreateConfigsArgs struct {
	Address       string
	TemplatesPath string
	FeFile        string
	FeTemplate    string
	BeFile        string
	BeTemplate    string
	ServiceName   string
	Monitor       bool // TODO: Not in use, remove
}

func (m Consul) PutService(address, instanceName string, r Registry) error {
	consulChannel := make(chan error)
	type data struct{ key, value string }
	d := []data{
		data{COLOR_KEY, r.ServiceColor},
		data{PATH_KEY, strings.Join(r.ServicePath, ",")},
		data{DOMAIN_KEY, r.ServiceDomain},
		data{PATH_TYPE_KEY, r.PathType},
		data{SKIP_CHECK_KEY, fmt.Sprintf("%t", r.SkipCheck)},
		data{CONSUL_TEMPLATE_FE_PATH_KEY, r.ConsulTemplateFePath},
		data{CONSUL_TEMPLATE_BE_PATH_KEY, r.ConsulTemplateBePath},
	}
	for _, e := range d {
		go m.SendPutRequest(address, r.ServiceName, e.key, e.value, instanceName, consulChannel)
	}
	go m.SendPutRequest(address, "service", r.ServiceName, "swarm", instanceName, consulChannel)
	for i := 0; i < len(d) + 1; i++ {
		err := <-consulChannel
		if err != nil {
			return fmt.Errorf("Could not send KV data to Consul\n%s", err.Error())
		}
	}
	return nil
}

func (m Consul) SendPutRequest(address, serviceName, key, value, instanceName string, c chan error) {
	c <- m.sendRequest("PUT", address, serviceName, key, value, instanceName)
}

func (m Consul) DeleteService(address, serviceName, instanceName string) error {
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	url := fmt.Sprintf("%s/v1/kv/%s/%s?recurse", address, instanceName, serviceName)
	client := &http.Client{}
	request, _ := http.NewRequest("DELETE", url, nil)
	_, err := client.Do(request)
	return err
}

func (m Consul) SendDeleteRequest(address, serviceName, key, value, instanceName string, c chan error) {
	c <- m.sendRequest("DELETE", address, serviceName, key, value, instanceName)
}

func (m Consul) CreateConfigs(args *CreateConfigsArgs) error {
	if err := m.createConfig(args.Address, args.TemplatesPath, args.FeFile, args.FeTemplate, args.ServiceName, "fe", false); err != nil {
		return err
	}
	if err := m.createConfig(
		args.Address,
		args.TemplatesPath,
		args.BeFile,
		args.BeTemplate,
		args.ServiceName,
		"be",
		args.Monitor,
	); err != nil {
		return err
	}
	return nil
}

func (m Consul) createConfig(address, templatesPath, file, template, serviceName, confType string, monitor bool) error {
	src := fmt.Sprintf("%s/%s", templatesPath, file)
	WriteConsulTemplateFile(src, []byte(template), 0664)
	dest := fmt.Sprintf("%s/%s-%s", templatesPath, serviceName, confType)
	if err := m.runConsulTemplateCmd(src, dest, address, monitor); err != nil {
		return fmt.Errorf("Could not create Consul configuration %s from the template %s\n%s", dest, src,  err.Error())
	}
	return nil
}

func (m Consul) sendRequest(requestType, address, serviceName, key, value, instanceName string) error {
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	url := fmt.Sprintf("%s/v1/kv/%s/%s/%s", address, instanceName, serviceName, key)
	client := &http.Client{}
	request, _ := http.NewRequest(requestType, url, strings.NewReader(value))
	_, err := client.Do(request)
	return err
}

func (m Consul) runConsulTemplateCmd(src, dest, address string, monitor bool) error {
	template := fmt.Sprintf(`%s:%s.cfg`, src, dest)
	cmdArgs := []string{
		"-consul", m.getConsulAddress(address),
		"-template", template,
		"-once",
	}
	cmd := exec.Command("consul-template", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmdRunConsulTemplate(cmd); err != nil {
		return fmt.Errorf("Command: %s\n%s\n", strings.Join(cmd.Args, " "), err.Error())
	}
	return nil
}

func (m Consul) getConsulAddress(address string) string {
	a := strings.ToLower(address)
	a = strings.TrimLeft(a, "http://")
	a = strings.TrimLeft(a, "https://")
	return a
}
