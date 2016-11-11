package server

import (
	"log"
	"net"
	"net/http"
)

var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var logPrintf = log.Printf
var lookupHost = net.LookupHost
