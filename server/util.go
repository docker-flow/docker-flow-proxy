package server

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var logPrintf = log.Printf
var lookupHost = net.LookupHost
var readFile = ioutil.ReadFile
