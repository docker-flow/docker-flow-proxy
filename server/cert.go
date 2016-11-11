package server

import (
	"../proxy"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var mu = &sync.Mutex{}

type Certer interface {
	Put(w http.ResponseWriter, req *http.Request) (string, error)
	GetAll(w http.ResponseWriter, req *http.Request) error
}

type Cert struct {
	ServiceName string
	CertsDir    string
	CertContent string
}

type CertResponse struct {
	Status  string
	Message string
	Certs   []Cert
}

func (m *Cert) GetAll(w http.ResponseWriter, req *http.Request) error {
	pCerts := proxy.Instance.GetCerts()
	certs := []Cert{}
	for name, content := range pCerts {
		cert := Cert{
			ServiceName: name,
			CertsDir:    "/certs",
			CertContent: content,
		}
		certs = append(certs, cert)
	}
	msg := CertResponse{
		Status:  "OK",
		Message: "",
		Certs:   certs,
	}
	m.writeOK(w, msg)
	return nil
}

func (m *Cert) Put(w http.ResponseWriter, req *http.Request) (string, error) {
	distribute, _ := strconv.ParseBool(req.URL.Query().Get("distribute"))
	if distribute {
		return "", m.sendDistributeRequests(w, req)
	}
	path, name, err := m.writeFile(w, req)
	if err != nil {
		return "", err
	}
	msg := CertResponse{
		Status:  "OK",
		Message: "",
	}
	m.writeOK(w, msg)
	proxy.Instance.AddCert(name)
	proxy.Instance.CreateConfigFromTemplates()
	logPrintf("Stored certificate %s", name)
	return path, nil
}

func (m *Cert) sendDistributeRequests(w http.ResponseWriter, req *http.Request) error {
	_, port, err := net.SplitHostPort(req.URL.Host)
	if err != nil {
		port = "8080"
	}
	status, err := server.SendDistributeRequests(req, port, m.ServiceName)
	if err != nil {
		return m.getError(w, err.Error(), err)
	} else if status >= 300 {
		msg := fmt.Sprintf("Distribution request failed with status %d", status)
		return m.getError(w, msg, fmt.Errorf(msg))
	}
	return nil
}

func (m *Cert) writeFile(w http.ResponseWriter, req *http.Request) (string, string, error) {
	name := req.URL.Query().Get("certName")
	if len(name) == 0 {
		return "", "", m.getError(w, "certName parameter is mandatory", fmt.Errorf("Query parameter certName is mandatory"))
	}
	defer func() { req.Body.Close() }()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", "", m.getError(w, err.Error(), err)
	} else if len(body) == 0 {
		return "", "", m.getError(w, "Body cannot be empty", fmt.Errorf("Body is empty"))
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Create(fmt.Sprintf("%s/%s", m.CertsDir, name))
	if err != nil {
		return "", "", m.getError(w, err.Error(), err)
	}
	f.Write(body)
	path, _ := filepath.Abs(fmt.Sprintf("%s/%s", m.CertsDir, name))
	return path, name, nil
}

func (m *Cert) writeOK(w http.ResponseWriter, msg interface{}) {
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	js, _ := json.Marshal(msg)
	w.Write(js)
}

func (m *Cert) getError(w http.ResponseWriter, message string, err error) error {
	w.WriteHeader(http.StatusBadRequest)
	js, _ := json.Marshal(CertResponse{
		Status:  "NOK",
		Message: message,
	})
	w.Write(js)
	return err
}

func NewCert(certsDir string) *Cert {
	return &Cert{
		CertsDir:    certsDir,
		ServiceName: os.Getenv("SERVICE_NAME"),
	}
}
