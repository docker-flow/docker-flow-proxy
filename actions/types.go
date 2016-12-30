package actions

type ServiceDest struct {
	// The internal port of a service that should be reconfigured.
	// The port is used only in the *swarm* mode.
	Port           string
	// The URL path of the service.
	ServicePath    []string
	// The source (entry) port of a service.
	// Useful only when specifying multiple destinations of a single service.
	SrcPort        int
	SrcPortAcl     string
	SrcPortAclName string
}

type ServiceReconfigure struct {
	// ACLs are ordered alphabetically by their names.
	// If not specified, serviceName is used instead.
	AclName              string
	// The path to the Consul Template representing a snippet of the backend configuration.
	// If set, proxy template will be loaded from the specified file.
	ConsulTemplateFePath string
	// The path to the Consul Template representing a snippet of the frontend configuration.
	// If specified, proxy template will be loaded from the specified file.
	ConsulTemplateBePath string
	// Whether to distribute a request to all the instances of the proxy.
	// Used only in the swarm mode.
	Distribute           bool
	// The internal HTTPS port of a service that should be reconfigured.
	// The port is used only in the swarm mode.
	// If not specified, the `port` parameter will be used instead.
	HttpsPort            int
	// The hostname where the service is running, for instance on a separate swarm.
	// If specified, the proxy will dispatch requests to that domain.
	OutboundHostname     string
	// The ACL derivative. Defaults to path_beg.
	// See https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path for more info.
	PathType             string
	// A regular expression to apply the modification.
	// If specified, `reqPathSearch` needs to be set as well.
	ReqPathReplace        string
	// A regular expression to search the content to be replaced.
	// If specified, `reqPathReplace` needs to be set as well.
	ReqPathSearch         string
	// Content of the PEM-encoded certificate to be used by the proxy when serving traffic over SSL.
	ServiceCert          string
	// The domain of the service.
	// If set, the proxy will allow access only to requests coming to that domain.
	ServiceDomain        []string
	// The name of the service.
	// It must match the name of the Swarm service or the one stored in Consul.
	ServiceName          string
	// The path to the template representing a snippet of the backend configuration.
	// If specified, the backend template will be loaded from the specified file.
	// If specified, `templateFePath` must be set as well.
	// See the https://github.com/vfarcic/docker-flow-proxy#templates section for more info.
	TemplateBePath       string
	// The path to the template representing a snippet of the frontend configuration.
	// If specified, the frontend template will be loaded from the specified file.
	// If specified, `templateBePath` must be set as well.
	// See the https://github.com/vfarcic/docker-flow-proxy#templates section for more info.
	TemplateFePath       string
	// Whether to skip adding proxy checks.
	// This option is used only in the default mode.
	SkipCheck            bool
	// A comma-separated list of credentials(<user>:<pass>) for HTTP basic auth, which applies only to the service that will be reconfigured.
	Users                []User
	// The default mode is designed to work with any setup and requires Consul and Registrator.
	// The swarm mode aims to leverage the benefits that come with Docker Swarm and new networking introduced in the 1.12 release.
	// The later mode (swarm) does not have any dependency but Docker Engine.
	// The swarm mode is recommended for all who use Docker Swarm features introduced in v1.12.
	Mode                 string `short:"m" long:"mode" env:"MODE" description:"If set to 'swarm', proxy will operate assuming that Docker service from v1.12+ is used."`
	ServiceColor         string
	ServicePort          string
	AclCondition         string
	FullServiceName      string
	Host                 string
	LookupRetry          int
	LookupRetryInterval  int
	ServiceDest          []ServiceDest
}
