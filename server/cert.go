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
	Init() error
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
	certName, certContent, err := m.getCertFromRequest(w, req)
	if err != nil {
		m.writeError(w, err)
	}
	path, err := m.writeFile(certName, certContent)
	if err != nil {
		m.writeError(w, fmt.Errorf("Query parameter certName is mandatory"))
		return "", err
	}
	msg := CertResponse{Status:  "OK", Message: ""}
	m.writeOK(w, msg)
	proxy.Instance.AddCert(certName)
	proxy.Instance.CreateConfigFromTemplates()
	logPrintf("Stored certificate %s", certName)
	return path, nil
}

func (m *Cert) Init() error {
	// TODO: get certs from all instances
	dns := fmt.Sprintf("tasks.%s", m.ServiceName)
	if _, err := lookupHost(dns); err == nil {
	}
	// TODO: Filter with results with the biggest certs collection

	// TODO: Change to names and content from other instances
//	dns := fmt.Sprintf("tasks.%s", serviceName)
	//	if ips, err := lookupHost(dns); err == nil {
	//
	//	}
//	m.writeFile("test.pem", []byte("THIS IS A CERTIFICATE"))
	return nil
}

func (m *Cert) getCertFromRequest(w http.ResponseWriter, req *http.Request) (certName string, certContent []byte, err error) {
	certName = req.URL.Query().Get("certName")
	if len(certName) == 0 {
		err := fmt.Errorf("Query parameter certName is mandatory")
		return "", []byte{}, err
	}
	defer func() { req.Body.Close() }()
	certContent, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return "", []byte{}, err
	} else if len(certContent) == 0 {
		err := fmt.Errorf("Body is empty")
		return "", []byte{}, err
	}
	return certName, certContent, nil
}

func (m *Cert) sendDistributeRequests(w http.ResponseWriter, req *http.Request) error {
	_, port, err := net.SplitHostPort(req.URL.Host)
	if err != nil {
		port = "8080"
	}
	status, err := server.SendDistributeRequests(req, port, m.ServiceName)
	if err != nil {
		return m.writeError(w, err)
	} else if status >= 300 {
		msg := fmt.Sprintf("Distribution request failed with status %d", status)
		return m.writeError(w, fmt.Errorf(msg))
	}
	return nil
}

func (m *Cert) writeFile(certName string, certContent []byte) (path string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if f, err := os.Create(fmt.Sprintf("%s/%s", m.CertsDir, certName)); err != nil {
		return "", err
	} else {
		f.Write(certContent)
	}
	path, _ = filepath.Abs(fmt.Sprintf("%s/%s", m.CertsDir, certName))
	return path, nil
}

func (m *Cert) writeOK(w http.ResponseWriter, msg interface{}) {
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	js, _ := json.Marshal(msg)
	w.Write(js)
}

func (m *Cert) writeError(w http.ResponseWriter, err error) error {
	w.WriteHeader(http.StatusBadRequest)
	js, _ := json.Marshal(CertResponse{
		Status:  "NOK",
		Message: err.Error(),
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
