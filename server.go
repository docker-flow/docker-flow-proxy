package main

import (
	"./actions"
	"./proxy"
	"./server"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"io/ioutil"
	"math/rand"
)

// TODO: Move to server package

const (
	DISTRIBUTED = "Distributed to all instances"
)

type Server interface {
	Execute(args []string) error
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type Serve struct {
	IP string `short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	// The default mode is designed to work with any setup and requires Consul and Registrator.
	// The swarm mode aims to leverage the benefits that come with Docker Swarm and new networking introduced in the 1.12 release.
	// The later mode (swarm) does not have any dependency but Docker Engine.
	// The swarm mode is recommended for all who use Docker Swarm features introduced in v1.12.
	Mode            string `short:"m" long:"mode" env:"MODE" description:"If set to 'swarm', proxy will operate assuming that Docker service from v1.12+ is used."`
	ListenerAddress string `short:"l" long:"listener-address" env:"LISTENER_ADDRESS" description:"The address of the Docker Flow: Swarm Listener. The address matches the name of the Swarm service (e.g. swarm-listener)"`
	Port            string `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	ServiceName     string `short:"n" long:"service-name" default:"proxy" env:"SERVICE_NAME" description:"The name of the proxy service. It is used only when running in 'swarm' mode and must match the '--name' parameter used to launch the service."`
	actions.BaseReconfigure
}

var serverImpl = Serve{}
var cert server.Certer = server.NewCert("/certs")
var reload actions.Reloader = actions.NewReload()
//exposed as global so can be changed in tests
var usersBasePath string = "/run/secrets/dfp_users_%s"

func (m *Serve) Execute(args []string) error {
	if proxy.Instance == nil {
		proxy.Instance = proxy.NewHaProxy(m.TemplatesPath, m.ConfigsPath)
	}
	logPrintf("Starting HAProxy")
	m.setConsulAddresses()
	NewRun().Execute([]string{})
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	lAddr := ""
	if len(m.ListenerAddress) > 0 {
		lAddr = fmt.Sprintf("http://%s:8080", m.ListenerAddress)
	}
	cert.Init()
	recon := actions.NewReconfigure(m.BaseReconfigure, proxy.Service{}, m.Mode)
	if err := recon.ReloadAllServices(
		m.ConsulAddresses,
		m.InstanceName,
		m.Mode,
		lAddr,
	); err != nil {
		return err
	}
	logPrintf(`Starting "Docker Flow: Proxy"`)
	if err := httpListenAndServe(address, m); err != nil {
		return err
	}
	return nil
}

func (m *Serve) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.EqualFold(req.URL.Path, "/v1/test") {
		logPrintf("Processing request %s", req.URL)
	}
	switch req.URL.Path {
	case "/v1/docker-flow-proxy/cert":
		if req.Method == "PUT" {
			cert.Put(w, req)
		} else {
			logPrintf("/v1/docker-flow-proxy/cert endpoint allows only PUT requests. Yours was %s", req.Method)
			w.WriteHeader(http.StatusNotFound)
		}
	case "/v1/docker-flow-proxy/certs":
		cert.GetAll(w, req)
	case "/v1/docker-flow-proxy/config":
		m.config(w, req)
	case "/v1/docker-flow-proxy/reconfigure":
		m.reconfigure(w, req)
	case "/v1/docker-flow-proxy/remove":
		m.remove(w, req)
	case "/v1/docker-flow-proxy/reload":
		m.reload(w, req)
	case "/v1/test", "/v2/test":
		js, _ := json.Marshal(server.Response{Status: "OK"})
		httpWriterSetContentType(w, "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	default:
		logPrintf("The endpoint %s is not supported", req.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *Serve) isValidReconf(service *proxy.Service) (bool, string) {
	if len(service.ServiceName) == 0 || len(service.ServiceDest) == 0 {
		return false, "serviceName parameter is mandatory"
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

func (m *Serve) isSwarm(mode string) bool {
	return strings.EqualFold("service", m.Mode) || strings.EqualFold("swarm", m.Mode)
}

func (m *Serve) hasPort(sd []proxy.ServiceDest) bool {
	return len(sd) > 0 && len(sd[0].Port) > 0
}

func (m *Serve) reload(w http.ResponseWriter, req *http.Request) {
	listenerAddr := ""
	recreate := m.getBoolParam(req, "recreate")
	if m.getBoolParam(req, "fromListener") {
		listenerAddr = m.ListenerAddress
	}
	reload.Execute(recreate, listenerAddr)
	w.WriteHeader(http.StatusOK)
	httpWriterSetContentType(w, "application/json")
	response := server.Response{
		Status: "OK",
	}
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m *Serve) reconfigure(w http.ResponseWriter, req *http.Request) {
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
	sr := m.getService(sd, req)
	response := server.Response{
		Mode:        m.Mode,
		Status:      "OK",
		ServiceName: sr.ServiceName,
		Service:     sr,
	}
	ok, msg := m.isValidReconf(&sr)
	if ok {
		if m.isSwarm(m.Mode) && !m.hasPort(sd) {
			m.writeBadRequest(w, &response, `When MODE is set to "service" or "swarm", the port query is mandatory`)
		} else if sr.Distribute {
			srv := server.Serve{}
			if status, err := srv.SendDistributeRequests(req, m.Port, m.ServiceName); err != nil || status >= 300 {
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
					cert.PutCert(sr.ServiceDomain[0], []byte(sr.ServiceCert))
				} else {
					cert.PutCert(sr.ServiceName, []byte(sr.ServiceCert))
				}
			}
			action := actions.NewReconfigure(m.BaseReconfigure, sr, m.Mode)
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

func getUsersFromString(serviceName, commaSeparatedUsers string, passEncrypted bool) ([]*proxy.User) {
	collectedUsers := []*proxy.User{}
	if len(commaSeparatedUsers) == 0 {
		return collectedUsers
	}
	users := strings.Split(commaSeparatedUsers, ",")
	for _, user := range users {
		user = strings.Trim(user, "\n\t ")
		if strings.Contains(user, ":") {
			userDetails := strings.Split(user, ":")
			if len(userDetails) != 2 || len(userDetails[0]) == 0 {
				logPrintf("For service %s there is an invalid user with no name or invalid format",
					serviceName)
			} else {
				collectedUsers = append(collectedUsers, &proxy.User{Username: userDetails[0], Password: userDetails[1], PassEncrypted: passEncrypted})
			}
		} else {
			if len(user) == 0 {
				logPrintf("For service %s there is an invalid user with no name or invalid format",
					serviceName)
			} else {
				collectedUsers = append(collectedUsers, &proxy.User{Username: user})
			}
		}
	}
	return collectedUsers
}

func getUsersFromFile(serviceName, fileName string, passEncrypted bool) ([]*proxy.User, error) {
	if len(fileName) > 0 {
		usersFile := fmt.Sprintf(usersBasePath, fileName)

		if content, err := ioutil.ReadFile(usersFile); err == nil {
			userContents := strings.TrimRight(string(content[:]), "\n")
			return getUsersFromString(serviceName,userContents, passEncrypted), nil
		} else {
			logPrintf("For service %s it was impossible to load userFile %s due to error %s",
				serviceName, usersFile, err.Error())
			return []*proxy.User{}, err
		}
	} else {
		return []*proxy.User{}, nil
	}
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

func mergeUsers(serviceName, usersParam, usersSecret string, usersPassEncrypted bool,
	globalUsersString string, globalUsersEncrypted bool) ([]proxy.User) {
	var collectedUsers []*proxy.User
	paramUsers := getUsersFromString(serviceName,usersParam, usersPassEncrypted)
	fileUsers, _ := getUsersFromFile(serviceName, usersSecret, usersPassEncrypted)
	if len(paramUsers) > 0 {
		if !allUsersHavePasswords(paramUsers) {
			if len(usersSecret) == 0 {
				fileUsers = getUsersFromString(serviceName, globalUsersString, globalUsersEncrypted)
			}
			for _, u := range paramUsers {
				if !u.HasPassword() {
					if userByName := findUserByName(fileUsers, u.Username); userByName != nil {
						u.Password = userByName.Password
						u.PassEncrypted = userByName.PassEncrypted
					} else {
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
		ret = append(ret, proxy.User{
			Username: "dummyUser",
			Password: strconv.FormatInt(rand.Int63(), 3)},
		)
	}
	if len(ret) == 0 {
		return nil
	}
	return ret

}

func (m *Serve) getService(sd []proxy.ServiceDest, req *http.Request) proxy.Service {
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
		globalUsersString, globalUsersEncrypted)
	return sr
}

func (m *Serve) getBoolParam(req *http.Request, param string) bool {
	value := false
	if len(req.URL.Query().Get(param)) > 0 {
		value, _ = strconv.ParseBool(req.URL.Query().Get(param))
	}
	return value
}

func (m *Serve) writeBadRequest(w http.ResponseWriter, resp *server.Response, msg string) {
	resp.Status = "NOK"
	resp.Message = msg
	w.WriteHeader(http.StatusBadRequest)
}

func (m *Serve) writeInternalServerError(w http.ResponseWriter, resp *server.Response, msg string) {
	resp.Status = "NOK"
	resp.Message = msg
	w.WriteHeader(http.StatusInternalServerError)
}

func (m *Serve) remove(w http.ResponseWriter, req *http.Request) {
	serviceName := req.URL.Query().Get("serviceName")
	distribute := m.getBoolParam(req, "distribute")
	response := server.Response{
		Status:      "OK",
		ServiceName: serviceName,
	}
	if distribute {
		response.Distribute = distribute
		response.Message = DISTRIBUTED
	}
	if len(serviceName) == 0 {
		response.Status = "NOK"
		response.Message = "The serviceName query is mandatory"
		w.WriteHeader(http.StatusBadRequest)
	} else if distribute {
		srv := server.Serve{}
		if status, err := srv.SendDistributeRequests(req, m.Port, m.ServiceName); err != nil || status >= 300 {
			m.writeInternalServerError(w, &response, err.Error())
		} else {
			response.Message = DISTRIBUTED
			w.WriteHeader(http.StatusOK)
		}
	} else {
		logPrintf("Processing remove request %s", req.URL.Path)
		aclName := req.URL.Query().Get("aclName")
		action := actions.NewRemove(
			serviceName,
			aclName,
			m.BaseReconfigure.ConfigsPath,
			m.BaseReconfigure.TemplatesPath,
			m.ConsulAddresses,
			m.InstanceName,
			m.Mode,
		)
		action.Execute([]string{})
		w.WriteHeader(http.StatusOK)
	}
	httpWriterSetContentType(w, "application/json")
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m *Serve) config(w http.ResponseWriter, req *http.Request) {
	httpWriterSetContentType(w, "text/html")
	out, err := proxy.Instance.ReadConfig()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte(out))
}

func (m *Serve) setConsulAddresses() {
	m.ConsulAddresses = []string{}
	if len(os.Getenv("CONSUL_ADDRESS")) > 0 {
		for _, address := range strings.Split(os.Getenv("CONSUL_ADDRESS"), ",") {
			if !strings.HasPrefix(address, "http") {
				address = fmt.Sprintf("http://%s", address)
			}
			m.ConsulAddresses = append(m.ConsulAddresses, address)
		}
	}
}
