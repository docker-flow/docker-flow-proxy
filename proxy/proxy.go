package proxy

var ProxyInstance Proxy = HaProxy{}

type Data struct {
	Certs     map[string]bool
	Services  map[string]Service
}

var data = Data{}

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates() error
	ReadConfig() (string, error)
	Reload() error
	AddCert(certName string)
	GetCerts() map[string]string
	AddService(service Service)
	RemoveService(service string)
}

// Mock
