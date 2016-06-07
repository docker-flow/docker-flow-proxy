package registry

const (
	COLOR_KEY                   = "color"
	PATH_KEY                    = "path"
	DOMAIN_KEY                  = "domain"
	PATH_TYPE_KEY               = "pathtype"
	SKIP_CHECK_KEY              = "skipcheck"
	CONSUL_TEMPLATE_FE_PATH_KEY = "consultemplatefepath"
	CONSUL_TEMPLATE_BE_PATH_KEY = "consultemplatebepath"
)

type Registry struct {
	ServiceName string
	ServiceColor string
	ServicePath []string
	ServiceDomain string
	PathType string
	SkipCheck bool
	ConsulTemplateFePath string
	ConsulTemplateBePath string
}

type Registrarable interface {
	PutService(address, instanceName string, r Registry) error
	SendPutRequest(address, serviceName, key, value, instanceName string, c chan error)
	DeleteService(address, serviceName, instanceName string) error
	SendDeleteRequest(address, serviceName, key, value, instanceName string, c chan error)
	CreateConfigs(args CreateConfigsArgs) error
}
