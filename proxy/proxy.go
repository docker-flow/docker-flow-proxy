package proxy

var ProxyInstance Proxy = HaProxy{}

type Data struct {
	Certs map[string]bool
}

var data = Data{}

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates() error
	ReadConfig() (string, error)
	Reload() error
	AddCert(certName string)
	GetCerts() map[string]string
}

// Mock
