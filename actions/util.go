package actions

import (
	"../registry"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

type Executable interface {
	Execute(args []string) error
}

func isSwarm(mode string) bool {
	return strings.EqualFold(mode, "service") || strings.EqualFold(mode, "swarm")
}

var lookupHost = net.LookupHost
var logPrintf = log.Printf
var httpGet = http.Get
var registryInstance registry.Registrarable = registry.Consul{}
var writeFeTemplate = ioutil.WriteFile
var writeBeTemplate = ioutil.WriteFile
var readTemplateFile = ioutil.ReadFile
