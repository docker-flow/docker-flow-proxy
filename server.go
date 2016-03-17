package main

import (
	"net/http"
	"fmt"
	"encoding/json"
)

type Serverable interface {
	Execute(args []string) error
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type Server struct {
	IP		string	`short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	Port	string  `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	BaseReconfigure
}


var server = Server{}

type Response struct {
	Status    	string
	Message 	string
	ServiceReconfigure
}

func (m Server) Execute(args []string) error {
	logPrintf("Starting HAProxy")
	NewRun().Execute([]string{})
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	logPrintf(`Starting "Docker Flow: Proxy"`)
	return httpListenAndServe(address, m)
}

func (m Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/v1/docker-flow-proxy/reconfigure":
		logPrintf("Processing request %s", req.URL)
		sr := ServiceReconfigure{
			ServiceName: req.URL.Query().Get("serviceName"),
			ServicePath: req.URL.Query().Get("servicePath"),
		}
		if len(sr.ServiceName) == 0 || len(sr.ServicePath) == 0 {
			js, _ := json.Marshal(Response{
				Status: "NOK",
				Message: "The following queries are mandatory: serviceName and servicePath",
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(js)
		} else {
			reconfig := NewReconfigure(
				m.BaseReconfigure,
				sr,
			)
			if err := reconfig.Execute([]string{}); err != nil {
				js, _ := json.Marshal(Response{
					Status: "NOK",
					Message: fmt.Sprintf("%#s", err),
				})
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(js)
			} else {
				js, _ := json.Marshal(Response{
					Status: "OK",
					ServiceReconfigure: sr,
				})
				httpWriterSetContentType(w, "application/json")
				w.Write(js)
			}
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
