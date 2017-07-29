package server

import (
	"../proxy"
	"encoding/json"
	"net/http"
	"strings"
)

type Configer interface {
	Get(w http.ResponseWriter, req *http.Request)
}

type Config struct{}

func NewConfig() Configer {
	return &Config{}
}

func (m *Config) Get(w http.ResponseWriter, req *http.Request) {
	status := http.StatusOK
	typeParam := req.URL.Query().Get("type")
	contentType := "text/html"
	body := []byte{}
	if strings.EqualFold(typeParam, "json") {
		contentType = "application/json"
		services := proxy.Instance.GetServices()
		body, _ = json.Marshal(services)
	} else {
		out, err := proxy.Instance.ReadConfig()
		if err != nil {
			status = http.StatusInternalServerError
		}
		body = []byte(out)
	}
	httpWriterSetContentType(w, contentType)
	w.WriteHeader(status)
	w.Write(body)
}
