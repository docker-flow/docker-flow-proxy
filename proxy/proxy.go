package proxy

var ProxyInstance Proxy = HaProxy{}

type Data struct {
	Services map[string]Service
}

var data = Data{}

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates() error
	ReadConfig() (string, error)
	Reload() error
	GetCertPaths() []string
	GetCerts() map[string]string
	AddService(service Service)
	RemoveService(service string)
}
