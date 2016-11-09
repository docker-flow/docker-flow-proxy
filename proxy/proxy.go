package proxy

var ProxyInstance Proxy = HaProxy{}

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates() error
	ReadConfig() (string, error)
	Reload() error
}
