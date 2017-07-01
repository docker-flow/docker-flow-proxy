package proxy

var proxyInstance proxy = HaProxy{}

type data struct {
	Services map[string]Service
}

var dataInstance = data{}

type proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates() error
	ReadConfig() (string, error)
	Reload() error
	GetCertPaths() []string
	GetCerts() map[string]string
	AddService(service Service)
	RemoveService(service string)
}
