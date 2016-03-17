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
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	return httpListenAndServe(address, m)
	return nil
}

func (m Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/v1/docker-flow-proxy/reconfigure":
		sr := ServiceReconfigure{
			ServiceName: req.URL.Query().Get("serviceName"),
			ServicePath: req.URL.Query().Get("servicePath"),
		}
		if len(sr.ServiceName) == 0 || len(sr.ServicePath) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			reconfig := NewReconfigure(
				m.BaseReconfigure,
				sr,
			)
			if err := reconfig.Execute([]string{}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
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
