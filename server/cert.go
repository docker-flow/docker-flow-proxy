package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"../proxy"
)

var mu = &sync.Mutex{}

type Certer interface {
	Put(w http.ResponseWriter, req *http.Request) (string, error)
}

type Cert struct {
	CertsDir string
}

type CertResponse struct {
	Status  string
	Message string
}

func (m Cert) Put(w http.ResponseWriter, req *http.Request) (string, error) {
	name := req.URL.Query().Get("certName")
	if len(name) == 0 {
		return "", fmt.Errorf("Query parameter certName is mandatory")
	}
	defer func() { req.Body.Close() }()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		js, _ := json.Marshal(CertResponse{
			Status:  "NOK",
			Message: err.Error(),
		})
		w.Write(js)
		w.WriteHeader(http.StatusBadRequest)
		return "", err
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Create(fmt.Sprintf("%s/%s", m.CertsDir, name))
	if err != nil {
		js, _ := json.Marshal(CertResponse{
			Status:  "NOK",
			Message: err.Error(),
		})
		w.Write(js)
		w.WriteHeader(http.StatusBadRequest)
		return "", err
	}
	f.Write(body)
	path, _ := filepath.Abs(fmt.Sprintf("%s/%s", m.CertsDir, name))
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	js, _ := json.Marshal(CertResponse{
		Status:  "OK",
		Message: "",
	})
	w.Write(js)
	proxy.Instance.AddCert(name)
	return path, nil
}

func NewCert(certsDir string) Cert {
	return Cert{
		CertsDir: certsDir,
	}
}
