package main

import (
	"io/ioutil"
	"os/exec"
	"net/http"
)

var readFile = ioutil.ReadFile
var readPidFile = ioutil.ReadFile
var readDir = ioutil.ReadDir
var cmdRunConsul = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var cmdRunHa = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var writeConsulTemplateFile = ioutil.WriteFile
var writeConsulConfigFile = ioutil.WriteFile
var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}