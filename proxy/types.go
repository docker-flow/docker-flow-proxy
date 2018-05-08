package proxy

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var usersBasePath string = "/run/secrets/dfp_users_%s"

// ServiceDest holds data used to generate proxy configuration. It is extracted as a separate struct since a single service can have multiple combinations.
type ServiceDest struct {
	// The list of allowed methods. If specified, a request with a method that is not on the list will be denied.
	AllowedMethods []string
	// HAProxy balance mode for in TCP groups.
	BalanceGroup string
	// Checks tcp connection. Only used in sni or tcp mode.
	CheckTCP bool
	// Enable sending of TCP keepalive packets on the client side. Only used in sni or tcp mode.
	Clitcpka bool
	// The list of denied methods. If specified, a request with a method that is on the list will be denied.
	DeniedMethods []string
	// Whether to deny HTTP requests thus allowing only HTTPS.
	DenyHttp bool
	// Whether to redirect all http requests to https
	HttpsOnly bool
	// HTTP code for HTTP to HTTPS redirects. This parameter is used only if `httpsOnly` is set to `true`.
	HttpsRedirectCode string
	// Whether to ignore authorization for this service destination.
	IgnoreAuthorization bool
	// The hostname where the service is running, for instance on a separate swarm.
	// If specified, the proxy will dispatch requests to that domain.
	OutboundHostname string
	// The internal port of a service that should be reconfigured.
	// The port is used only in the *swarm* mode.
	Port string
	// If a request is sent to one of the domains in this list, it will be redirected to one of the values of the `ServiceDomain`.
	RedirectFromDomain []string
	// The request mode. The proxy should be able to work with any mode supported by HAProxy.
	// However, actively supported and tested modes are *http*, *tcp*, and *sni*.
	ReqMode string
	// Internal use only. Do not modify.
	ReqModeFormatted string
	// A regular expression to search and replace the content.
	// Search and replace values are separated with comma (`,`).
	// Multiple search/replace combinations can be separated with colon (`:`).
	// This field deprecates `ReqPathSearch` and `ReqPathReplace`.
	ReqPathSearchReplace string
	// The domain of the service.
	// If set, the proxy will allow access only to requests coming to that domain.
	ServiceDomain []string
	// Name of service group used by tcp groups
	ServiceGroup string
	// Headers used to filter requests
	ServiceHeader map[string]string
	// The URL path of the service.
	ServicePath []string
	// The URL path that should be excluded from the rules.
	ServicePathExclude []string
	// The source (entry) port of a service.
	// Useful only when specifying multiple destinations of a single service.
	SrcPort int
	// Internal use only. Do not modify.
	SrcPortAcl string
	// Internal use only. Do not modify.
	SrcPortAclName string
	// If set to true, server certificates are not verified. This flag should be set for SSL enabled backend services.
	SslVerifyNone bool
	// The server timeout in seconds
	TimeoutServer string
	// The client timeout in seconds
	TimeoutClient string
	// The tunnel timeout in seconds
	TimeoutTunnel string
	// Whether to verify client SSL and deny request when it is invalid
	VerifyClientSsl bool
	// If specified, only requests with the same agent will be forwarded to the backend.
	UserAgent UserAgent
	// User defined value.
	// This value is not used with current template.
	// It is designed as a way to provide additional data that can be used with custom templates.
	UserDef string
	// Internal use only
	Index int
	// Internal use only
	ReqPathSearchReplaceFormatted []string
}

// UserAgent holds data used to generate proxy configuration. It is extracted as a separate struct since each user agent needs an ACL identifier. If specified, only requests with the same agent will be forwarded to the backend.
type UserAgent struct {
	Value   []string
	AclName string
}

// Service contains description of a service that should be added to the proxy configuration.
type Service struct {
	// ACLs are ordered alphabetically by their names.
	// If not specified, serviceName is used instead.
	AclName string `split_words:"true"`
	// Additional headers that will be added to the request before forwarding it to the service.
	// Please consult https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html#add-a-header-to-the-request for more info.
	AddReqHeader []string `split_words:"true"`
	// Additional headers that will be added to the response before forwarding it to the client.
	AddResHeader []string `split_words:"true"`
	// Additional configuration that will be added to the bottom of the service backend
	BackendExtra string `split_words:"true"`
	// Enable resolvers.
	// Provides higher reliability at the cost of backend initialization time.
	// If enabled, it might take a few seconds until a backend is resolved and operational.
	// Resolvers can be customized through the environment variable RESOLVERS.
	CheckResolvers bool `split_words:"true"`
	// Enable HTTP compression.
	// The currently supported algorithms are: identity, gzip, deflate, raw-deflate.
	CompressionAlgo string `split_words:"true"`
	// The type of files that will be compressed.
	CompressionType string `split_words:"true"`
	// One of the five connection modes supported by the HAProxy.
	// `http-keep-alive`: all requests and responses are processed.
	// `http-tunnel`: only the first request and response are processed, everything else is forwarded with no analysis.
	// `httpclose`: tunnel with "Connection: close" added in both directions.
	// `http-server-close`: the server-facing connection is closed after the response.
	// `forceclose`: the connection is actively closed after end of response.
	// In general, it is preferred to use http-server-close with application servers, and some static servers might benefit from http-keep-alive.
	// Connection mode is restricted to HTTP mode only.
	// If specified, connection mode will be applied to the backend section.
	ConnectionMode string `split_words:"true"`
	// Internal use only
	Debug bool
	// Internal use only
	DebugFormat string
	// Additional headers that will be deleted in the request before forwarding it to the service. Please consult https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html#delete-a-header-in-the-request for more info.
	DelReqHeader []string `split_words:"true"`
	// Additional headers that will be deleted in the response before forwarding it to the client. Please consult https://www.haproxy.com/doc/aloha/7.0/haproxy/http_rewriting.html#delete-a-header-in-the-response for more info.
	DelResHeader []string `split_words:"true"`
	// The type of service discovery.
	// Currently supported are Overlay (default) and DNS.
	// Overlay discovery relies on Overlay network to perform round-robing load balancing. This is the recommended discovery.
	// DNS discovery detects the replicas using DNS SD.
	DiscoveryType string
	// Whether to distribute a request to all the instances of the proxy.
	// Used only in the swarm mode.
	Distribute bool `split_words:"true"`
	// The internal HTTPS port of a service that should be reconfigured.
	// The port is used only in the swarm mode.
	// If not specified, the `port` parameter will be used instead.
	HttpsPort int `split_words:"true"`
	// If set to true, it will be the default_backend service.
	IsDefaultBackend bool `split_words:"true"`
	// The ACL derivative. Defaults to path_beg.
	// See https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#7.3.6-path for more info.
	PathType string `split_words:"true"`
	// Whether to redirect to https when X-Forwarded-Proto is http
	RedirectWhenHttpProto bool `split_words:"true"`
	// The number of replicas of a service.
	// This parameter is currently used only if `DiscoveryType` is set to `DNS`.
	Replicas int `split_words:"true"`
	// TODO: Deprecated since Dec. 2017.
	// A regular expression to apply the modification.
	// If specified, `reqPathSearch` needs to be set as well.
	ReqPathReplace string `split_words:"true"`
	// TODO: Deprecated since Dec. 2017.
	// A regular expression to search the content to be replaced.
	// If specified, `reqPathReplace` needs to be set as well.
	ReqPathSearch string `split_words:"true"`
	// Content of the PEM-encoded certificate to be used by the proxy when serving traffic over SSL.
	ServiceCert string `split_words:"true"`
	// The algorithm that should be applied to domain acl. The default value is `hdr_beg(host)`.
	ServiceDomainAlgo string
	// The name of the service.
	// It must match the name of the Swarm service.
	ServiceName string `split_words:"true"`
	// Determines the type of sticky sessions. If set to `sticky-server`, session cookie will be set by the proxy. Any other value means that sticky sessions are not used and load balancing is performed by Docker's Overlay network. Please open an issue if you'd like support for other types of sticky sessions.
	SessionType string `split_words:"true"`
	// Additional headers that will be set to the request before forwarding it to the service. If a specified header exists, it will be replaced with the new one.
	SetReqHeader []string `split_words:"true"`
	// Additional headers that will be set to the response before forwarding it to the client. If a specified header exists, it will be replaced with the new one.
	SetResHeader []string `split_words:"true"`
	// The path to the template representing a snippet of the backend configuration.
	// If specified, the backend template will be loaded from the specified file.
	// If specified, `templateFePath` must be set as well.
	// See the https://github.com/docker-flow/docker-flow-proxy#templates section for more info.
	TemplateBePath string `split_words:"true"`
	// The path to the template representing a snippet of the frontend configuration.
	// If specified, the frontend template will be loaded from the specified file.
	// If specified, `templateBePath` must be set as well.
	// See the https://github.com/docker-flow/docker-flow-proxy#templates section for more info.
	TemplateFePath string `split_words:"true"`
	// Internal use only.
	UseGlobalUsers bool
	// A comma-separated list of credentials(<user>:<pass>) for HTTP basic auth, which applies only to the service that will be reconfigured.
	Users []User `split_words:"true"`
	// The rest of variables are for internal use only
	ServicePort         string
	AclCondition        string
	LookupRetry         int
	LookupRetryInterval int
	ServiceDest         []ServiceDest
	Tasks               []string
}

// Services contains the list of services used inside the proxy
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
}

func (slice Services) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
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

func extractUsersFromString(context, usersString string, encrypted, skipEmptyPassword bool) []*User {
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

// ServiceParameterProvider defines common interface for translating parameters into structs.
type ServiceParameterProvider interface {
	Fill(service *Service)
	GetString(name string) string
}

// GetServiceFromMap returns Service struct by extracting request parameters
func GetServiceFromMap(req *map[string]string) *Service {
	provider := mapParameterProvider{theMap: req}
	return GetServiceFromProvider(&provider)
}

// GetServiceFromProvider returns Service by extracting parameters from ServiceParameterProvider
func GetServiceFromProvider(provider ServiceParameterProvider) *Service {
	sr := new(Service)
	provider.Fill(sr)
	separator := os.Getenv("SEPARATOR")
	// TODO: Remove. It's added to maintain backwards compatibility with the deprecated parameter serviceDomainMatchAll (since July 2017)
	if strings.EqualFold(provider.GetString("serviceDomainMatchAll"), "true") {
		sr.ServiceDomainAlgo = "hdr_dom(host)"
	}
	if len(provider.GetString("httpsPort")) > 0 {
		sr.HttpsPort, _ = strconv.Atoi(provider.GetString("httpsPort"))
	}
	if len(provider.GetString("addReqHeader")) > 0 {
		sr.AddReqHeader = strings.Split(provider.GetString("addReqHeader"), separator)
	} else if len(provider.GetString("addHeader")) > 0 { // TODO: Deprecated since Apr. 2017.
		sr.AddReqHeader = strings.Split(provider.GetString("addHeader"), separator)
	}
	if len(provider.GetString("setReqHeader")) > 0 {
		sr.SetReqHeader = strings.Split(provider.GetString("setReqHeader"), separator)
	} else if len(provider.GetString("setHeader")) > 0 { // TODO: Deprecated since Apr. 2017.
		sr.SetReqHeader = strings.Split(provider.GetString("setHeader"), separator)
	}
	if len(provider.GetString("delReqHeader")) > 0 {
		sr.DelReqHeader = strings.Split(provider.GetString("delReqHeader"), separator)
	}
	if len(provider.GetString("addResHeader")) > 0 {
		sr.AddResHeader = strings.Split(provider.GetString("addResHeader"), separator)
	}
	if len(provider.GetString("setResHeader")) > 0 {
		sr.SetResHeader = strings.Split(provider.GetString("setResHeader"), separator)
	}
	if len(provider.GetString("delResHeader")) > 0 {
		sr.DelResHeader = strings.Split(provider.GetString("delResHeader"), separator)
	}
	if len(sr.SessionType) > 0 {
		sr.Tasks, _ = lookupHost("tasks." + sr.ServiceName)
	}
	globalUsersString := getSecretOrEnvVar("USERS", "")
	globalUsersEncrypted := strings.EqualFold(getSecretOrEnvVar("USERS_PASS_ENCRYPTED", ""), "true")
	sr.Users = mergeUsers(
		sr.ServiceName,
		provider.GetString("users"),
		provider.GetString("usersSecret"),
		getBoolParam(provider, "usersPassEncrypted", ""),
		globalUsersString,
		globalUsersEncrypted,
	)

	sr.ServiceDest = getServiceDestList(sr, provider)
	return sr
}

func getServiceDestList(sr *Service, provider ServiceParameterProvider) []ServiceDest {
	sdList := []ServiceDest{}
	sd := getServiceDest(sr, provider, -1)
	httpsOnly := sd.HttpsOnly
	if !httpsOnly {
		httpsOnly, _ = strconv.ParseBool(os.Getenv("HTTPS_ONLY"))
	}
	for i := 1; i <= 10; i++ {
		sd := getServiceDest(sr, provider, i)
		if isServiceDestValid(provider, i) {
			sdList = append(sdList, sd)
		} else {
			break
		}
	}
	if len(sdList) == 0 && isServiceDestValid(provider, -1) {
		sdList = append(sdList, sd)
	}
	if len(sdList) == 0 {
		reqMode := getFromString(provider, "reqMode", "")
		if len(reqMode) == 0 {
			reqMode = "http"
		}
		sdList = append(sdList, ServiceDest{ReqMode: reqMode})
	}
	for i, sd := range sdList {
		if len(sd.ServiceDomain) > 0 && len(sd.ServicePath) == 0 {
			sdList[i].ServicePath = []string{"/"}
		}
		if httpsOnly && !sd.HttpsOnly {
			sdList[i].HttpsOnly = true
		}
	}
	return sdList
}

func getServiceDest(sr *Service, provider ServiceParameterProvider, index int) ServiceDest {
	separator := os.Getenv("SEPARATOR")
	suffix := ""
	if index > 0 {
		suffix = fmt.Sprintf(".%d", index)
	}
	userAgent := UserAgent{}
	userAgentString := getFromString(provider, "userAgent", suffix)
	if len(userAgentString) > 0 {
		userAgent.Value = strings.Split(userAgentString, separator)
		userAgent.AclName = replaceNonAlphabetAndNumbers(userAgent.Value)
	}
	reqMode := getFromString(provider, "reqMode", suffix)
	if len(reqMode) == 0 {
		reqMode = "http"
	}
	srcPort, _ := strconv.Atoi(getFromString(provider, "srcPort", suffix))
	headerString := getFromString(provider, "serviceHeader", suffix)
	header := map[string]string{}
	if len(headerString) > 0 {
		for _, value := range strings.Split(headerString, separator) {
			values := strings.Split(value, ":")
			if len(values) == 2 {
				header[strings.Trim(values[0], " ")] = strings.Trim(values[1], " ")
			}
		}
	}
	sdIndex := index
	if sdIndex < 0 {
		sdIndex = 0
	}
	reqPathSearchReplaceFormatted := []string{}
	if len(sr.ReqPathSearch) > 0 {
		reqPathSearchReplaceFormatted = append(
			reqPathSearchReplaceFormatted,
			fmt.Sprintf("%s,%s", sr.ReqPathSearch, sr.ReqPathReplace),
		)
	}
	reqPathSearchReplace := getFromString(provider, "reqPathSearchReplace", suffix)
	if len(reqPathSearchReplace) > 0 {
		searchReplace := strings.Split(reqPathSearchReplace, ":")
		reqPathSearchReplaceFormatted = append(
			reqPathSearchReplaceFormatted,
			searchReplace...,
		)
	}
	return ServiceDest{
		AllowedMethods:                getSliceFromString(provider, "allowedMethods", suffix),
		BalanceGroup:                  getFromString(provider, "balanceGroup", suffix),
		CheckTCP:                      getBoolParam(provider, "checkTcp", suffix),
		Clitcpka:                      getBoolParam(provider, "clitcpka", suffix),
		DeniedMethods:                 getSliceFromString(provider, "deniedMethods", suffix),
		DenyHttp:                      getBoolParam(provider, "denyHttp", suffix),
		HttpsOnly:                     getBoolParam(provider, "httpsOnly", suffix),
		HttpsRedirectCode:             getFromString(provider, "httpsRedirectCode", suffix),
		IgnoreAuthorization:           getBoolParam(provider, "ignoreAuthorization", suffix),
		OutboundHostname:              getFromString(provider, "outboundHostname", suffix),
		Port:                          getFromString(provider, "port", suffix),
		RedirectFromDomain:            getSliceFromString(provider, "redirectFromDomain", suffix),
		ReqMode:                       reqMode,
		ReqPathSearchReplace:          reqPathSearchReplace,
		ReqPathSearchReplaceFormatted: reqPathSearchReplaceFormatted,
		ServiceDomain:                 getSliceFromString(provider, "serviceDomain", suffix),
		ServiceGroup:                  getFromString(provider, "serviceGroup", suffix),
		ServiceHeader:                 header,
		ServicePath:                   getSliceFromString(provider, "servicePath", suffix),
		ServicePathExclude:            getSliceFromString(provider, "servicePathExclude", suffix),
		SrcPort:                       srcPort,
		SslVerifyNone:                 getBoolParam(provider, "sslVerifyNone", suffix),
		TimeoutClient:                 getFromString(provider, "timeoutClient", suffix),
		TimeoutServer:                 getFromString(provider, "timeoutServer", suffix),
		TimeoutTunnel:                 getFromString(provider, "timeoutTunnel", suffix),
		VerifyClientSsl:               getBoolParam(provider, "verifyClientSsl", suffix),
		UserAgent:                     userAgent,
		UserDef:                       getFromString(provider, "userDef", suffix),
		Index:                         sdIndex,
	}
}

func getSliceFromString(provider ServiceParameterProvider, param, index string) []string {
	separator := os.Getenv("SEPARATOR")
	key := fmt.Sprintf("%s%s", param, index)
	if len(provider.GetString(key)) > 0 {
		return strings.Split(provider.GetString(key), separator)
	} else if len(provider.GetString(param)) > 0 {
		return strings.Split(provider.GetString(param), separator)
	}
	return []string{}
}

func getFromString(provider ServiceParameterProvider, param, index string) string {
	key := fmt.Sprintf("%s%s", param, index)
	value := provider.GetString(key)
	if len(value) > 0 {
		return value
	}
	return provider.GetString(param)
}

func isServiceDestValid(provider ServiceParameterProvider, index int) bool {
	suffix := ""
	if index > 0 {
		suffix = fmt.Sprintf(".%d", index)
	}
	hasPath := len(provider.GetString(fmt.Sprintf("servicePath%s", suffix))) > 0
	hasPort := len(provider.GetString(fmt.Sprintf("port%s", suffix))) > 0
	hasDomain := len(provider.GetString(fmt.Sprintf("serviceDomain%s", suffix))) > 0
	return hasPath || hasPort || hasDomain
}

func getBoolParam(req ServiceParameterProvider, param, index string) bool {
	value := false
	key := fmt.Sprintf("%s%s", param, index)
	if len(req.GetString(key)) > 0 {
		value, _ = strconv.ParseBool(req.GetString(key))
	} else if len(req.GetString(param)) > 0 {
		value, _ = strconv.ParseBool(req.GetString(param))
	}
	return value
}

func mergeUsers(
	serviceName,
	usersParam,
	usersSecret string,
	usersPassEncrypted bool,
	globalUsersString string,
	globalUsersEncrypted bool,
) []User {
	var collectedUsers []*User
	paramUsers := extractUsersFromString(serviceName, usersParam, usersPassEncrypted, false)
	fileUsers, _ := getUsersFromFile(serviceName, usersSecret, usersPassEncrypted)
	if len(paramUsers) > 0 {
		if !allUsersHavePasswords(paramUsers) {
			if len(usersSecret) == 0 {
				fileUsers = extractUsersFromString(serviceName, globalUsersString, globalUsersEncrypted, true)
			}
			for _, u := range paramUsers {
				if !u.hasPassword() {
					if userByName := findUserByName(fileUsers, u.Username); userByName != nil {
						u.Password = "sdasdsad"
						u.Password = userByName.Password
						u.PassEncrypted = userByName.PassEncrypted
					} else {
						// TODO: Return an error
						// TODO: Test
						logPrintf("For service %s it was impossible to find password for user %s.",
							serviceName, u.Username)
					}
				}
			}
		}
		collectedUsers = paramUsers
	} else {
		collectedUsers = fileUsers
	}
	ret := []User{}
	for _, u := range collectedUsers {
		if u.hasPassword() {
			ret = append(ret, *u)
		}
	}
	if len(ret) == 0 && (len(usersParam) != 0 || len(usersSecret) != 0) {
		//we haven't found any users but they were requested so generating dummy one
		ret = append(ret, *randomUser())
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func getUsersFromFile(serviceName, fileName string, passEncrypted bool) ([]*User, error) {
	if len(fileName) > 0 {
		usersFile := fmt.Sprintf(usersBasePath, fileName)
		content, err := readFile(usersFile)
		if err != nil {
			logPrintf(
				"For service %s it was impossible to load userFile %s due to error %s",
				serviceName,
				usersFile,
				err.Error(),
			)
			return []*User{}, err
		}
		userContents := strings.TrimRight(string(content[:]), "\n")
		return extractUsersFromString(serviceName, userContents, passEncrypted, true), nil
	}
	return []*User{}, nil
}

func allUsersHavePasswords(users []*User) bool {
	for _, u := range users {
		if !u.hasPassword() {
			return false
		}
	}
	return true
}

func findUserByName(users []*User, name string) *User {
	for _, u := range users {
		if strings.EqualFold(name, u.Username) {
			return u
		}
	}
	return nil
}
