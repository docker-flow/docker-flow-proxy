package main

import (
	"./registry"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var readPidFile = ioutil.ReadFile
var readConfigsDir = ioutil.ReadDir
var readConfigsFile = ioutil.ReadFile
var readTemplateFile = ioutil.ReadFile
var cmdRunHa = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var writeFile = ioutil.WriteFile
var writeFeTemplate = ioutil.WriteFile
var writeBeTemplate = ioutil.WriteFile
var osRemove = os.Remove
var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var logPrintf = log.Printf

type Executable interface {
	Execute(args []string) error
}

var registryInstance registry.Registrarable = registry.Consul{}
