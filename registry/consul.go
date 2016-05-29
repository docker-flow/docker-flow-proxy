package registry
import (
	"fmt"
	"net/http"
	"strings"
)

type Consul struct {}

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
	for i := 0; i < len(d); i++ {
		err := <-consulChannel
		if err != nil {
			return fmt.Errorf("Could not send data to Consul\n%s", err.Error())
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
