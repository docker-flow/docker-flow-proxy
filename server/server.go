package server

import (
	"../proxy"
	"../actions"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"encoding/json"
)

var usersBasePath string = "/run/secrets/dfp_users_%s"
var extractUsersFromString = proxy.ExtractUsersFromString
var reload actions.Reloader = actions.NewReload()

type Server interface {
	GetServiceFromUrl(sd []proxy.ServiceDest, req *http.Request) proxy.Service
	TestHandler(w http.ResponseWriter, req *http.Request)
	ReloadHandler(w http.ResponseWriter, req *http.Request)
	RemoveHandler(w http.ResponseWriter, req *http.Request)
}

const (
	DISTRIBUTED = "Distributed to all instances"
)

type Serve struct{
	ListenerAddress string
	Mode            string
	Port            string
	ServiceName     string
	ConfigsPath     string
	TemplatesPath   string
	ConsulAddresses []string
	Cert            Certer
}

func NewServer(listenerAddr, mode, port, serviceName, configsPath, templatesPath string, consulAddresses []string, cert Certer) *Serve {
	return &Serve{
		ListenerAddress: listenerAddr,
		Mode:            mode,
		Port:            port,
		ServiceName:     serviceName,
		ConfigsPath:     configsPath,
		TemplatesPath:   templatesPath,
		ConsulAddresses: consulAddresses,
		Cert:            cert,
	}
}

type Response struct {
	Mode        string
	Status      string
	Message     string
	ServiceName string
	proxy.Service
}

// TODO: Refactor to mux schema
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

func (m *Serve) TestHandler(w http.ResponseWriter, req *http.Request) {
	js, _ := json.Marshal(Response{Status: "OK"})
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

func (m *Serve) ReconfigureHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	params := new(ReconfigureParams)
	decoder.Decode(params, req.Form)
	path := []string{}
	if len(req.URL.Query().Get("servicePath")) > 0 {
		path = strings.Split(req.URL.Query().Get("servicePath"), ",")
	}
	port := req.URL.Query().Get("port")
	srcPort, _ := strconv.Atoi(req.URL.Query().Get("srcPort"))
	sd := []proxy.ServiceDest{}
	ctmplFePath := req.URL.Query().Get("consulTemplateFePath")
	ctmplBePath := req.URL.Query().Get("consulTemplateBePath")
	if len(path) > 0 || len(port) > 0 || (len(ctmplFePath) > 0 && len(ctmplBePath) > 0) {
		sd = append(
			sd,
			proxy.ServiceDest{Port: port, SrcPort: srcPort, ServicePath: path},
		)
	}
	for i := 1; i <= 10; i++ {
		port := req.URL.Query().Get(fmt.Sprintf("port.%d", i))
		path := req.URL.Query().Get(fmt.Sprintf("servicePath.%d", i))
		srcPort, _ := strconv.Atoi(req.URL.Query().Get(fmt.Sprintf("srcPort.%d", i)))
		if len(path) > 0 && len(port) > 0 {
			sd = append(
				sd,
				proxy.ServiceDest{Port: port, SrcPort: srcPort, ServicePath: strings.Split(path, ",")},
			)
		} else {
			break
		}
	}
	sr := m.GetServiceFromUrl(sd, req)
	response := Response{
		Mode:        m.Mode,
		Status:      "OK",
		ServiceName: params.ServiceName,
		Service:     sr,
	}
	ok, msg := m.isValidReconf(&sr)
	if ok {
		if m.isSwarm(m.Mode) && !m.hasPort(sd) {
			m.writeBadRequest(w, &response, `When MODE is set to "service" or "swarm", the port query is mandatory`)
		} else if params.Distribute {
			if status, err := SendDistributeRequests(req, m.Port, m.ServiceName); err != nil || status >= 300 {
				m.writeInternalServerError(w, &response, err.Error())
			} else {
				response.Message = DISTRIBUTED
				w.WriteHeader(http.StatusOK)
			}
		} else {
			if len(sr.ServiceCert) > 0 {
				// Replace \n with proper carriage return as new lines are not supported in labels
				sr.ServiceCert = strings.Replace(sr.ServiceCert, "\\n", "\n", -1)
				if len(sr.ServiceDomain) > 0 {
					m.Cert.PutCert(sr.ServiceDomain[0], []byte(sr.ServiceCert))
				} else {
					m.Cert.PutCert(sr.ServiceName, []byte(sr.ServiceCert))
				}
			}
			br := actions.BaseReconfigure {
				ConsulAddresses: m.ConsulAddresses,
				ConfigsPath: m.ConfigsPath,
				InstanceName: m.ServiceName,
				TemplatesPath: m.TemplatesPath,
			}
			action := actions.NewReconfigure(br, sr, m.Mode)
			if err := action.Execute([]string{}); err != nil {
				m.writeInternalServerError(w, &response, err.Error())
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	} else {
		m.writeBadRequest(w, &response, msg)
	}
	httpWriterSetContentType(w, "application/json")
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m *Serve) ReloadHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	params := new(ReloadParams)
	decoder.Decode(params, req.Form)
	listenerAddr := ""
	if params.FromListener {
		listenerAddr = m.ListenerAddress
	}
	reload.Execute(params.Recreate, listenerAddr)
	w.WriteHeader(http.StatusOK)
	httpWriterSetContentType(w, "application/json")
	response := Response{
		Status: "OK",
	}
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m *Serve) RemoveHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	params := new(RemoveParams)
	decoder.Decode(params, req.Form)
	header := http.StatusOK
	response := Response{
		Status:      "OK",
		ServiceName: params.ServiceName,
	}
	if params.Distribute {
		response.Distribute = params.Distribute
		response.Message = DISTRIBUTED
	}
	if len(params.ServiceName) == 0 {
		response.Status = "NOK"
		response.Message = "The serviceName query is mandatory"
		header = http.StatusBadRequest
	} else if params.Distribute {
		if status, err := SendDistributeRequests(req, m.Port, m.ServiceName); err != nil || status >= 300 {
			response.Status = "NOK"
			response.Message = err.Error()
			header = http.StatusInternalServerError
		}
	} else {
		logPrintf("Processing remove request %s", req.URL.Path)
		action := actions.NewRemove(
			params.ServiceName,
			params.AclName,
			m.ConfigsPath,
			m.TemplatesPath,
			m.ConsulAddresses,
			m.ServiceName,
			m.Mode,
		)
		action.Execute([]string{})
	}
	w.WriteHeader(header)
	httpWriterSetContentType(w, "application/json")
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m *Serve) writeBadRequest(w http.ResponseWriter, resp *Response, msg string) {
	resp.Status = "NOK"
	resp.Message = msg
	w.WriteHeader(http.StatusBadRequest)
}

func (m *Serve) writeInternalServerError(w http.ResponseWriter, resp *Response, msg string) {
	resp.Status = "NOK"
	resp.Message = msg
	w.WriteHeader(http.StatusInternalServerError)
}

func (m *Serve) isSwarm(mode string) bool {
	return strings.EqualFold("service", m.Mode) || strings.EqualFold("swarm", m.Mode)
}

func (m *Serve) hasPort(sd []proxy.ServiceDest) bool {
	return len(sd) > 0 && len(sd[0].Port) > 0
}

func (m *Serve) isValidReconf(service *proxy.Service) (bool, string) {
	if len(service.ServiceName) == 0 {
		return false, "serviceName parameter is mandatory"
	} else if len(service.ServiceDest) == 0 {
		return false, "There must be at least one destination"
	}
	hasPath := len(service.ServiceDest[0].ServicePath) > 0
	hasSrcPort := service.ServiceDest[0].SrcPort > 0
	hasPort := len(service.ServiceDest[0].Port) > 0
	if strings.EqualFold(service.ReqMode, "http") {
		if !hasPath && len(service.ConsulTemplateFePath) == 0 {
			return false, "When using reqMode http, servicePath or (consulTemplateFePath and consulTemplateBePath) are mandatory"
		}
	} else if !hasSrcPort || !hasPort {
		return false, "When NOT using reqMode http (e.g. tcp), srcPort and port parameters are mandatory."
	}
	return true, ""
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
			return proxy.ExtractUsersFromString(serviceName, userContents, passEncrypted, true), nil
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
