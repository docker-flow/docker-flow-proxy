package main

import (
	"io/ioutil"
	"os/exec"
	"net/http"
	"log"
	"os"
)

var readPidFile = ioutil.ReadFile
var readConfigsDir = ioutil.ReadDir
var readConfigsFile = ioutil.ReadFile
var cmdRunConsul = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var cmdRunHa = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var writeFile = ioutil.WriteFile
var writeConsulTemplateFile = ioutil.WriteFile
var osRemove = os.Remove
var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var logPrintf = log.Printf

type Executable interface {
	Execute(args []string) error
}