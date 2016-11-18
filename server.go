package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	DISTRIBUTED = "Distributed to all instances"
)

type Server interface {
	Execute(args []string) error
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type Serve struct {
	IP          string `short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	Mode        string `short:"m" long:"mode" env:"MODE" description:"If set to 'swarm', proxy will operate assuming that Docker service from v1.12+ is used."`
	ListenerAddress string `short:"l" long:"listener-address" env:"LISTENER_ADDRESS" description:"The address of the Docker Flow: Swarm Listener. The address matches the name of the Swarm service (e.g. swarm-listener)"`
	Port        string `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	ServiceName string `short:"n" long:"service-name" default:"proxy" env:"SERVICE_NAME" description:"The name of the proxy service. It is used only when running in 'swarm' mode and must match the '--name' parameter used to launch the service."`
	BaseReconfigure
}

var server = Serve{}

type Response struct {
	Status               string
	Message              string
	ServiceName          string
	ServiceColor         string
	ServicePath          []string
	ServiceDomain        string
	OutboundHostname     string
	ConsulTemplateFePath string
	ConsulTemplateBePath string
	PathType             string
	SkipCheck            bool
	Mode                 string
	Port                 string
	Distribute           bool
}

func (m *Serve) Execute(args []string) error {
	logPrintf("Starting HAProxy")
	m.setConsulAddresses()
	NewRun().Execute([]string{})
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	recon := NewReconfigure(m.BaseReconfigure, ServiceReconfigure{})
	lAddr := ""
	if len(m.ListenerAddress) > 0 {
		lAddr = fmt.Sprintf("http://%s:8080", m.ListenerAddress)
	}
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
	case "/v1/docker-flow-proxy/reconfigure":
		m.reconfigure(w, req)
	case "/v1/docker-flow-proxy/remove":
		m.remove(w, req)
	case "/v1/docker-flow-proxy/config":
		m.config(w, req)
	case "/v1/test", "/v2/test":
		js, _ := json.Marshal(Response{Status: "OK"})
		httpWriterSetContentType(w, "application/json")
		if !strings.EqualFold(req.URL.Path, "/v1/test") {
			logPrintf("Invoked %s", req.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *Serve) SendDistributeRequests(req *http.Request, serviceName string) (status int, err error) {
	values := req.URL.Query()
	values.Set("distribute", "false")
	req.URL.RawQuery = values.Encode()
	dns := fmt.Sprintf("tasks.%s", m.ServiceName)
	failedDns := []string{}
	if ips, err := lookupHost(dns); err == nil {
		for i := 0; i < len(ips); i++ {
			req.URL.Host = fmt.Sprintf("%s:%s", ips[i], m.Port)
			client := &http.Client{}
			addr := fmt.Sprintf("http://%s:%s%s?%s", ips[i], m.Port, req.URL.Path, req.URL.RawQuery)
			logPrintf("Sending distribution request to %s", addr)
			if resp, err := client.Get(addr); err != nil || resp.StatusCode >= 300 {
				if err != nil {
					logPrintf(err.Error())
				}
				failedDns = append(failedDns, ips[i])
			}
		}
	} else {
		return http.StatusBadRequest, fmt.Errorf("Could not perform DNS %s lookup", dns)
	}
	if len(failedDns) > 0 {
		return http.StatusBadRequest, fmt.Errorf("Could not send distribute request to the following addresses: %s", failedDns)
	}
	return http.StatusOK, err
}

func (m *Serve) isValidReconf(name string, path []string, templateFePath string) bool {
	return len(name) > 0 && (len(path) > 0 || len(templateFePath) > 0)
}

func (m *Serve) reconfigure(w http.ResponseWriter, req *http.Request) {
	sr := ServiceReconfigure{
		ServiceName:          req.URL.Query().Get("serviceName"),
		ServiceColor:         req.URL.Query().Get("serviceColor"),
		ServiceDomain:        req.URL.Query().Get("serviceDomain"),
		OutboundHostname:     req.URL.Query().Get("outboundHostname"),
		ConsulTemplateFePath: req.URL.Query().Get("consulTemplateFePath"),
		ConsulTemplateBePath: req.URL.Query().Get("consulTemplateBePath"),
		PathType:             req.URL.Query().Get("pathType"),
		Port:                 req.URL.Query().Get("port"),
		Mode:                 m.Mode,
	}
	if len(req.URL.Query().Get("servicePath")) > 0 {
		sr.ServicePath = strings.Split(req.URL.Query().Get("servicePath"), ",")
	}
	if len(req.URL.Query().Get("skipCheck")) > 0 {
		sr.SkipCheck, _ = strconv.ParseBool(req.URL.Query().Get("skipCheck"))
	}
	if len(req.URL.Query().Get("distribute")) > 0 {
		sr.Distribute, _ = strconv.ParseBool(req.URL.Query().Get("distribute"))
	}
	response := Response{
		Status:               "OK",
		ServiceName:          sr.ServiceName,
		ServiceColor:         sr.ServiceColor,
		ServicePath:          sr.ServicePath,
		ServiceDomain:        sr.ServiceDomain,
		OutboundHostname:     sr.OutboundHostname,
		ConsulTemplateFePath: sr.ConsulTemplateFePath,
		ConsulTemplateBePath: sr.ConsulTemplateBePath,
		PathType:             sr.PathType,
		SkipCheck:            sr.SkipCheck,
		Mode:                 sr.Mode,
		Port:                 sr.Port,
		Distribute:           sr.Distribute,
	}
	if m.isValidReconf(sr.ServiceName, sr.ServicePath, sr.ConsulTemplateFePath) {
		if (strings.EqualFold("service", m.Mode) || strings.EqualFold("swarm", m.Mode)) && len(sr.Port) == 0 {
			m.writeBadRequest(w, &response, `When MODE is set to "service" or "swarm", the port query is mandatory`)
		} else if sr.Distribute {
			if status, err := m.SendDistributeRequests(req, sr.ServiceName); err != nil || status >= 300 {
				m.writeInternalServerError(w, &response, err.Error())
			} else {
				response.Message = DISTRIBUTED
				w.WriteHeader(http.StatusOK)
			}
		} else {
			action := NewReconfigure(m.BaseReconfigure, sr)
			if err := action.Execute([]string{}); err != nil {
				m.writeInternalServerError(w, &response, err.Error())
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	} else {
		m.writeBadRequest(w, &response, "The following queries are mandatory: (serviceName and servicePath) or (serviceName, consulTemplateFePath, and consulTemplateBePath)")
	}
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

func (m *Serve) remove(w http.ResponseWriter, req *http.Request) {
	serviceName := req.URL.Query().Get("serviceName")
	distribute := false
	response := Response{
		Status:      "OK",
		ServiceName: serviceName,
	}
	if len(req.URL.Query().Get("distribute")) > 0 {
		distribute, _ = strconv.ParseBool(req.URL.Query().Get("distribute"))
		if distribute {
			response.Distribute = distribute
			response.Message = DISTRIBUTED
		}
	}
	if len(serviceName) == 0 {
		response.Status = "NOK"
		response.Message = "The serviceName query is mandatory"
		w.WriteHeader(http.StatusBadRequest)
	} else if distribute {
		if status, err := m.SendDistributeRequests(req, serviceName); err != nil || status >= 300 {
			m.writeInternalServerError(w, &response, err.Error())
		} else {
			response.Message = DISTRIBUTED
			w.WriteHeader(http.StatusOK)
		}
	} else {
		logPrintf("Processing remove request %s", req.URL.Path)
		action := NewRemove(
			serviceName,
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
	out, err := proxy.ReadConfig(m.BaseReconfigure.ConfigsPath)
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
