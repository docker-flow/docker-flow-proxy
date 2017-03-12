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
	"github.com/gorilla/mux"
)

// TODO: Move to server package

// TODO: Remove
const (
	DISTRIBUTED = "Distributed to all instances"
)

type Server interface {
	Execute(args []string) error
}

type Serve struct {
	IP string `short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	// TODO: Remove
	// The default mode is designed to work with any setup and requires Consul and Registrator.
	// The swarm mode aims to leverage the benefits that come with Docker Swarm and new networking introduced in the 1.12 release.
	// The later mode (swarm) does not have any dependency but Docker Engine.
	// The swarm mode is recommended for all who use Docker Swarm features introduced in v1.12.
	Mode            string `short:"m" long:"mode" env:"MODE" description:"If set to 'swarm', proxy will operate assuming that Docker service from v1.12+ is used."`
	ListenerAddress string `short:"l" long:"listener-address" env:"LISTENER_ADDRESS" description:"The address of the Docker Flow: Swarm Listener. The address matches the name of the Swarm service (e.g. swarm-listener)"`
	Port            string `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	ServiceName     string `short:"n" long:"service-name" default:"proxy" env:"SERVICE_NAME" description:"The name of the proxy service. It is used only when running in 'swarm' mode and must match the '--name' parameter used to launch the service."`
	// TODO: Remove
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
	r := mux.NewRouter().StrictSlash(true)
	var server2 = server.NewServer(
		m.ListenerAddress,
		m.Mode,
		m.Port,
		m.ServiceName,
		m.ConfigsPath,
		m.TemplatesPath,
		m.ConsulAddresses,
	)
	r.HandleFunc("/v1/docker-flow-proxy/cert", m.CertPutHandler).Methods("PUT")
	r.HandleFunc("/v1/docker-flow-proxy/certs", m.CertsHandler)
	r.HandleFunc("/v1/docker-flow-proxy/config", m.ConfigHandler)
	r.HandleFunc("/v1/docker-flow-proxy/reconfigure", m.ReconfigureHandler)
	r.HandleFunc("/v1/docker-flow-proxy/remove", server2.RemoveHandler)
	r.HandleFunc("/v1/docker-flow-proxy/reload", server2.ReloadHandler)
	r.HandleFunc("/v1/test", server2.TestHandler)
	r.HandleFunc("/v2/test", server2.TestHandler)
	if err := httpListenAndServe(address, r); err != nil {
		return err
	}
	return nil
}

// TODO: Move to server package
func (m *Serve) CertPutHandler(w http.ResponseWriter, req *http.Request) {
	cert.Put(w, req)
}

// TODO: Move to server package
func (m *Serve) CertsHandler(w http.ResponseWriter, req *http.Request) {
	cert.GetAll(w, req)
}

// TODO: Move to server package
func (m *Serve) ConfigHandler(w http.ResponseWriter, req *http.Request) {
	m.config(w, req)
}

// TODO: Move to server package
func (m *Serve) ReconfigureHandler(w http.ResponseWriter, req *http.Request) {
	m.reconfigure(w, req)
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
	srv := server.Serve{}
	sr := srv.GetServiceFromUrl(sd, req)
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
			if status, err := server.SendDistributeRequests(req, m.Port, m.ServiceName); err != nil || status >= 300 {
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
