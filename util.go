package main

import (
	"./registry"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

var readTemplateFile = ioutil.ReadFile
var readFile = ioutil.ReadFile
var writeFeTemplate = ioutil.WriteFile
var writeBeTemplate = ioutil.WriteFile
var osRemove = os.Remove
var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var httpGet = http.Get
var logPrintf = log.Printf

type Executable interface {
	Execute(args []string) error
}

func isSwarm(mode string) bool {
	return strings.EqualFold(mode, "service") || strings.EqualFold(mode, "swarm")
}

var lookupHost = net.LookupHost
var mu = &sync.Mutex{}
var registryInstance registry.Registrarable = registry.Consul{}
