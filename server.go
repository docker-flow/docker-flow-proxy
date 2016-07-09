package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Server interface {
	Execute(args []string) error
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type Serve struct {
	IP   string `short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	Port string `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	Mode string `short:"m" long:"mode" env:"MODE" description:"If set to service, proxy will operate assuming that Docker service from v1.12+ is used."`
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
	ConsulTemplateFePath string
	ConsulTemplateBePath string
	PathType             string
	SkipCheck            bool
	Mode                 string
	Port                 string
}

func (m Serve) Execute(args []string) error {
	logPrintf("Starting HAProxy")
	NewRun().Execute([]string{})
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	if !strings.EqualFold(m.Mode, "service") && !strings.EqualFold(m.Mode, "swarm") {
		if err := NewReconfigure(
			m.BaseReconfigure,
			ServiceReconfigure{},
		).ReloadAllServices(m.ConsulAddress, m.InstanceName); err != nil {
			return err
		}
	}
	logPrintf(`Starting "Docker Flow: Proxy"`)
	if err := httpListenAndServe(address, m); err != nil {
		return err
	}
	return nil
}

func (m Serve) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logPrintf("Processing request %s", req.URL)
	switch req.URL.Path {
	case "/v1/docker-flow-proxy/reconfigure":
		m.reconfigure(w, req)
	case "/v1/docker-flow-proxy/remove":
		m.remove(w, req)
	case "/v1/test", "/v2/test":
		js, _ := json.Marshal(Response{Status: "OK"})
		httpWriterSetContentType(w, "application/json")
		logPrintf("Invoked %s", req.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m Serve) isValidReconf(name string, path []string, templateFePath string) bool {
	return len(name) > 0 && (len(path) > 0 || len(templateFePath) > 0)
}

func (m Serve) reconfigure(w http.ResponseWriter, req *http.Request) {
	sr := ServiceReconfigure{
		ServiceName:          req.URL.Query().Get("serviceName"),
		ServiceColor:         req.URL.Query().Get("serviceColor"),
		ServiceDomain:        req.URL.Query().Get("serviceDomain"),
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
	response := Response{
		Status:               "OK",
		ServiceName:          sr.ServiceName,
		ServiceColor:         sr.ServiceColor,
		ServicePath:          sr.ServicePath,
		ServiceDomain:        sr.ServiceDomain,
		ConsulTemplateFePath: sr.ConsulTemplateFePath,
		ConsulTemplateBePath: sr.ConsulTemplateBePath,
		PathType:             sr.PathType,
		SkipCheck:            sr.SkipCheck,
		Mode:                 sr.Mode,
		Port:                 sr.Port,
	}
	if m.isValidReconf(sr.ServiceName, sr.ServicePath, sr.ConsulTemplateFePath) {
		if (strings.EqualFold("service", m.Mode) || strings.EqualFold("swarm", m.Mode)) && len(sr.Port) == 0 {
			response.Status = "NOK"
			response.Message = `When MODE is set to "service", the port query is mandatory`
			w.WriteHeader(http.StatusBadRequest)
		} else {
			action := NewReconfigure(
				m.BaseReconfigure,
				sr,
			)
			if err := action.Execute([]string{}); err != nil {
				response.Status = "NOK"
				response.Message = fmt.Sprintf("%s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.WriteHeader(http.StatusOK)
		}
	} else {
		response.Status = "NOK"
		response.Message = "The following queries are mandatory: (serviceName and servicePath) or (serviceName, consulTemplateFePath, and consulTemplateBePath)"
		w.WriteHeader(http.StatusBadRequest)
	}
	httpWriterSetContentType(w, "application/json")
	js, _ := json.Marshal(response)
	w.Write(js)
}

func (m Serve) remove(w http.ResponseWriter, req *http.Request) {
	serviceName := req.URL.Query().Get("serviceName")
	response := Response{
		Status:      "OK",
		ServiceName: serviceName,
	}
	if len(serviceName) == 0 {
		response.Status = "NOK"
		response.Message = "The serviceName query is mandatory"
		w.WriteHeader(http.StatusBadRequest)
	} else {
		logPrintf("Processing remove request %s", req.URL.Path)
		action := NewRemove(
			serviceName,
			m.BaseReconfigure.ConfigsPath,
			m.BaseReconfigure.TemplatesPath,
			m.ConsulAddress,
			m.InstanceName,
			m.Mode,
		)
		action.Execute([]string{})
	}
	httpWriterSetContentType(w, "application/json")
	js, _ := json.Marshal(response)
	w.Write(js)
}
