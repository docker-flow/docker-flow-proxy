package server

import (
	"../proxy"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"strconv"
)

var server Server = NewServer()
var usersBasePath string = "/run/secrets/dfp_users_%s"
var extractUsersFromString = proxy.ExtractUsersFromString

type Server interface {
	SendDistributeRequests(req *http.Request, port, proxyServiceName string) (status int, err error)
	GetServiceFromUrl(sd []proxy.ServiceDest, req *http.Request) proxy.Service
}

type Serve struct{}

func NewServer() *Serve {
	return &Serve{}
}

type Response struct {
	Mode        string
	Status      string
	Message     string
	ServiceName string
	proxy.Service
}

func (m *Serve) SendDistributeRequests(req *http.Request, port, proxyServiceName string) (status int, err error) {
	values := req.URL.Query()
	values.Set("distribute", "false")
	req.URL.RawQuery = values.Encode()
	dns := fmt.Sprintf("tasks.%s", proxyServiceName)
	failedDns := []string{}
	method := req.Method
	body := ""
	if req.Body != nil {
		defer func() { req.Body.Close() }()
		reqBody, _ := ioutil.ReadAll(req.Body)
		body = string(reqBody)
	}
	if ips, err := lookupHost(dns); err == nil {
		for i := 0; i < len(ips); i++ {
			req.URL.Host = fmt.Sprintf("%s:%s", ips[i], port)
			client := &http.Client{}
			addr := fmt.Sprintf("http://%s:%s%s?%s", ips[i], port, req.URL.Path, req.URL.RawQuery)
			logPrintf("Sending distribution request to %s", addr)
			req, _ := http.NewRequest(method, addr, strings.NewReader(body))
			if resp, err := client.Do(req); err != nil || resp.StatusCode >= 300 {
				failedDns = append(failedDns, ips[i])
			}
		}
	} else {
		return http.StatusBadRequest, fmt.Errorf("Could not perform DNS %s lookup. If the proxy is not called 'proxy', you must set SERVICE_NAME=<name-of-the-proxy>.", dns)
	}
	if len(failedDns) > 0 {
		return http.StatusBadRequest, fmt.Errorf("Could not send distribute request to the following addresses: %s", failedDns)
	}
	return http.StatusOK, err
}

func (m *Serve) GetServiceFromUrl(sd []proxy.ServiceDest, req *http.Request) proxy.Service {
	sr := proxy.Service{
		ServiceDest:          sd,
		ServiceName:          req.URL.Query().Get("serviceName"),
		AclName:              req.URL.Query().Get("aclName"),
		ServiceColor:         req.URL.Query().Get("serviceColor"),
		ServiceCert:          req.URL.Query().Get("serviceCert"),
		OutboundHostname:     req.URL.Query().Get("outboundHostname"),
		ConsulTemplateFePath: req.URL.Query().Get("consulTemplateFePath"),
		ConsulTemplateBePath: req.URL.Query().Get("consulTemplateBePath"),
		PathType:             req.URL.Query().Get("pathType"),
		ReqRepSearch:         req.URL.Query().Get("reqRepSearch"),  // TODO: Deprecated (dec. 2016).
		ReqRepReplace:        req.URL.Query().Get("reqRepReplace"), // TODO: Deprecated (dec. 2016).
		ReqPathSearch:        req.URL.Query().Get("reqPathSearch"),
		ReqPathReplace:       req.URL.Query().Get("reqPathReplace"),
		TemplateFePath:       req.URL.Query().Get("templateFePath"),
		TemplateBePath:       req.URL.Query().Get("templateBePath"),
		TimeoutServer:        req.URL.Query().Get("timeoutServer"),
		TimeoutTunnel:        req.URL.Query().Get("timeoutTunnel"),
	}
	if len(req.URL.Query().Get("reqMode")) > 0 {
		sr.ReqMode = req.URL.Query().Get("reqMode")
	} else {
		sr.ReqMode = "http"
	}
	sr.HttpsOnly = m.getBoolParam(req, "httpsOnly")
	sr.XForwardedProto = m.getBoolParam(req, "xForwardedProto")
	sr.RedirectWhenHttpProto = m.getBoolParam(req, "redirectWhenHttpProto")
	if len(req.URL.Query().Get("httpsPort")) > 0 {
		sr.HttpsPort, _ = strconv.Atoi(req.URL.Query().Get("httpsPort"))
	}
	if len(req.URL.Query().Get("serviceDomain")) > 0 {
		sr.ServiceDomain = strings.Split(req.URL.Query().Get("serviceDomain"), ",")
	}
	sr.SkipCheck = m.getBoolParam(req, "skipCheck")
	sr.Distribute = m.getBoolParam(req, "distribute")
	sr.SslVerifyNone = m.getBoolParam(req, "sslVerifyNone")
	sr.ServiceDomainMatchAll = m.getBoolParam(req, "serviceDomainMatchAll")
	globalUsersString := proxy.GetSecretOrEnvVar("USERS", "")
	globalUsersEncrypted := strings.EqualFold(proxy.GetSecretOrEnvVar("USERS_PASS_ENCRYPTED", ""), "true")
	sr.Users = mergeUsers(sr.ServiceName,
		req.URL.Query().Get("users"),
		req.URL.Query().Get("usersSecret"),
		m.getBoolParam(req, "usersPassEncrypted"),
		globalUsersString,
		globalUsersEncrypted,
	)
	return sr
}

func (m *Serve) getBoolParam(req *http.Request, param string) bool {
	value := false
	if len(req.URL.Query().Get(param)) > 0 {
		value, _ = strconv.ParseBool(req.URL.Query().Get(param))
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
) []proxy.User {
	var collectedUsers []*proxy.User
	paramUsers := extractUsersFromString(serviceName, usersParam, usersPassEncrypted, false)
	fileUsers, _ := getUsersFromFile(serviceName, usersSecret, usersPassEncrypted)
	if len(paramUsers) > 0 {
		if !allUsersHavePasswords(paramUsers) {
			if len(usersSecret) == 0 {
				fileUsers = proxy.ExtractUsersFromString(serviceName, globalUsersString, globalUsersEncrypted, true)
			}
			for _, u := range paramUsers {
				if !u.HasPassword() {
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
	ret := []proxy.User{}
	for _, u := range collectedUsers {
		if u.HasPassword() {
			ret = append(ret, *u)
		}
	}
	if len(ret) == 0 && (len(usersParam) != 0 || len(usersSecret) != 0) {
		//we haven't found any users but they were requested so generating dummy one
		ret = append(ret, *proxy.RandomUser())
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func getUsersFromFile(serviceName, fileName string, passEncrypted bool) ([]*proxy.User, error) {
	if len(fileName) > 0 {
		usersFile := fmt.Sprintf(usersBasePath, fileName)
		if content, err := readFile(usersFile); err == nil {
			userContents := strings.TrimRight(string(content[:]), "\n")
			return proxy.ExtractUsersFromString(serviceName,userContents, passEncrypted, true), nil
		} else { // TODO: Test
			logPrintf("For service %s it was impossible to load userFile %s due to error %s",
				serviceName, usersFile, err.Error())
			return []*proxy.User{}, err
		}
	}
	return []*proxy.User{}, nil
}

func allUsersHavePasswords(users []*proxy.User) bool {
	for _, u := range users {
		if !u.HasPassword() {
			return false
		}
	}
	return true
}

func findUserByName(users []*proxy.User, name string) *proxy.User {
	for _, u := range users {
		if strings.EqualFold(name, u.Username) {
			return u
		}
	}
	return nil
}