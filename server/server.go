package server

import (
	"../proxy"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var server Server = NewServer()

type Server interface {
	SendDistributeRequests(req *http.Request, port, proxyServiceName string) (status int, err error)
}

type Serve struct{}

func NewServer() *Serve {
	return &Serve{}
}

type ServiceDest struct {
	Port    string
	Path    []string
	SrcPort int
}

type Response struct {
	Status               string
	Message              string
	ServiceName          string
	AclName              string
	ServiceColor         string
	ServiceDomain        []string
	ServiceCert          string
	OutboundHostname     string
	ConsulTemplateFePath string
	ConsulTemplateBePath string
	PathType             string
	SkipCheck            bool
	Mode                 string
	HttpsPort            int
	Distribute           bool
	Users                []proxy.User
	ReqRepSearch         string // TODO: Deprecated (dec. 2016).
	ReqRepReplace        string // TODO: Deprecated (dec. 2016).
	ReqPathSearch         string
	ReqPathReplace        string
	TemplateFePath       string
	TemplateBePath       string
	ServiceDest          []ServiceDest
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
