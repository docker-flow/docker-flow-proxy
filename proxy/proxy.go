package proxy

var ProxyInstance Proxy = HaProxy{}

type Proxy interface {
	RunCmd(extraArgs []string) error
	CreateConfigFromTemplates(templatesPath string, configsPath string) error
	ReadConfig(configsPath string) (string, error)
	Reload() error
}

