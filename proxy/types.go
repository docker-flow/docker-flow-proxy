package proxy

import (
	"math/rand"
	"strconv"
	"strings"
)

type ServiceDest struct {
	// The internal port of a service that should be reconfigured.
	// The port is used only in the *swarm* mode.
	Port string
	// The URL path of the service.
	ServicePath []string
	// The source (entry) port of a service.
	// Useful only when specifying multiple destinations of a single service.
	SrcPort        int
	SrcPortAcl     string
	SrcPortAclName string
}

type Service struct {
	// ACLs are ordered alphabetically by their names.
	// If not specified, serviceName is used instead.
	AclName string `split_words:"true"`
	// The path to the Consul Template representing a snippet of the backend configuration.
	// If set, proxy template will be loaded from the specified file.
	ConsulTemplateBePath string `split_words:"true"`
	// The path to the Consul Template representing a snippet of the frontend configuration.
	// If specified, proxy template will be loaded from the specified file.
	ConsulTemplateFePath string `split_words:"true"`
	// Whether to distribute a request to all the instances of the proxy.
	// Used only in the swarm mode.
	Distribute bool `split_words:"true"`
	// Whether to redirect all http requests to https
	HttpsOnly bool `split_words:"true"`
	// The internal HTTPS port of a service that should be reconfigured.
	// The port is used only in the swarm mode.
	// If not specified, the `port` parameter will be used instead.
	HttpsPort int `split_words:"true"`
	// The hostname where the service is running, for instance on a separate swarm.
	// If specified, the proxy will dispatch requests to that domain.
	OutboundHostname string `split_words:"true"`
	// The ACL derivative. Defaults to path_beg.
	// See https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path for more info.
	PathType string `split_words:"true"`
	// Whether to redirect to https when X-Forwarded-Proto is http
	RedirectWhenHttpProto bool `split_words:"true"`
	// The request mode. The proxy should be able to work with any mode supported by HAProxy. However, actively supported and tested modes are *http* and *tcp*. Please open an GitHub issue if the mode you're using does not work as expected. The default value is *http*.
	// Adding support for *sni*. Setting this to "sni" implies TCP with an SNI-based routing.
	ReqMode string `default:"http" split_words:"true"`
	// Deprecated in favor of ReqPathReplace
	ReqRepReplace string `split_words:"true"`
	// Deprecated in favor of ReqPathSearch
	ReqRepSearch string `split_words:"true"`
	// A regular expression to apply the modification.
	// If specified, `reqPathSearch` needs to be set as well.
	ReqPathReplace string `split_words:"true"`
	// A regular expression to search the content to be replaced.
	// If specified, `reqPathReplace` needs to be set as well.
	ReqPathSearch string `split_words:"true"`
	// Content of the PEM-encoded certificate to be used by the proxy when serving traffic over SSL.
	ServiceCert string `split_words:"true"`
	// The domain of the service.
	// If set, the proxy will allow access only to requests coming to that domain.
	ServiceDomain []string `split_words:"true"`
	// Whether to include subdomains and FDQN domains in the match. If set to false, and, for example, `serviceDomain` is set to `acme.com`, `something.acme.com` would not be considered a match unless this parameter is set to `true`. If this option is used, it is recommended to put any subdomains higher in the list using `aclName`.
	ServiceDomainMatchAll bool `split_words:"true"`
	// The name of the service.
	// It must match the name of the Swarm service or the one stored in Consul.
	ServiceName string `split_words:"true"`
	// Whether to skip adding proxy checks.
	// This option is used only in the default mode.
	SkipCheck bool `split_words:"true"`
	// If set to true, server certificates are not verified. This flag should be set for SSL enabled backend services.
	SslVerifyNone bool `split_words:"true"`
	// The path to the template representing a snippet of the backend configuration.
	// If specified, the backend template will be loaded from the specified file.
	// If specified, `templateFePath` must be set as well.
	// See the https://github.com/vfarcic/docker-flow-proxy#templates section for more info.
	TemplateBePath string `split_words:"true"`
	// The path to the template representing a snippet of the frontend configuration.
	// If specified, the frontend template will be loaded from the specified file.
	// If specified, `templateBePath` must be set as well.
	// See the https://github.com/vfarcic/docker-flow-proxy#templates section for more info.
	TemplateFePath string `split_words:"true"`
	// The server timeout in seconds
	TimeoutServer string `split_words:"true"`
	// The tunnel timeout in seconds
	TimeoutTunnel string `split_words:"true"`
	// A comma-separated list of credentials(<user>:<pass>) for HTTP basic auth, which applies only to the service that will be reconfigured.
	Users []User `split_words:"true"`
	// Whether to add "X-Forwarded-Proto https" header.
	XForwardedProto     bool `envconfig:"x_forwarded_proto" split_words:"true"`
	ServiceColor        string
	ServicePort         string
	AclCondition        string
	FullServiceName     string
	Host                string
	LookupRetry         int
	LookupRetryInterval int
	ServiceDest         []ServiceDest

}

type Services []Service

func (slice Services) Len() int {
	return len(slice)
}

func (slice Services) Less(i, j int) bool {
	firstHasRoot := hasRoot(slice[i])
	secondHasRoot := hasRoot(slice[j])
	firstHasWellKnown := hasWellKnown(slice[i])
	secondHasWellKnown := hasWellKnown(slice[j])
	if firstHasWellKnown && !secondHasWellKnown {
		return true
	} else if !firstHasWellKnown && secondHasWellKnown {
		return false
	} else if firstHasRoot && !secondHasRoot {
		return false
	} else if !firstHasRoot && secondHasRoot {
		return true
	} else {
		return slice[i].AclName < slice[j].AclName
	}
	return true
}

func hasRoot(service Service) bool {
	for _, sd := range service.ServiceDest {
		for _, path := range sd.ServicePath {
			if path == "/" {
				return true
			}
		}
	}
	return false
}

func hasWellKnown(service Service) bool {
	for _, sd := range service.ServiceDest {
		for _, path := range sd.ServicePath {
			if strings.HasPrefix(strings.ToLower(path), "/.well-known") {
				return true
			}
		}
	}
	return false
}

func (slice Services) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type User struct {
	Username      string
	Password      string
	PassEncrypted bool
}

func (user *User) HasPassword() bool {
	return len(user.Password) > 0
}

func RandomUser() *User {
	return &User{
		Username:      "dummyUser",
		PassEncrypted: true,
		Password:      strconv.FormatInt(rand.Int63(), 3)}
}

func ExtractUsersFromString(context, usersString string, encrypted, skipEmptyPassword bool) []*User {
	collectedUsers := []*User{}
	// TODO: Test
	if len(usersString) == 0 {
		return collectedUsers
	}
	splitter := func(x rune) bool {
		return x == '\n' || x == ','
	}
	users := strings.FieldsFunc(usersString, splitter)
	for _, user := range users {
		user = strings.Trim(user, "\n\t ")
		if len(user) == 0 {
			continue
		}
		if strings.Contains(user, ":") {
			colonIndex := strings.Index(user, ":")
			userName := strings.Trim(user[0:colonIndex], "\t ")
			userPass := strings.Trim(user[colonIndex+1:], "\t ")
			if len(userName) == 0 || len(userPass) == 0 {
				logPrintf("There is a user with no name or with invalid format for the service %s", context)
			} else {
				collectedUsers = append(collectedUsers, &User{Username: userName, Password: userPass, PassEncrypted: encrypted})
			}
		} else {
			if len(user) == 0 { // TODO: Test
				logPrintf("There is a user with no name or with invalid format for the service %s", context)
			} else if skipEmptyPassword { // TODO: Test
				logPrintf(
					"For service %s There is a user %s with no password for the service %s",
					user,
					context,
				)
			} else if !skipEmptyPassword {
				collectedUsers = append(collectedUsers, &User{Username: user})
			}
		}
	}
	return collectedUsers
}
