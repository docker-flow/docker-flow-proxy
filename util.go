package main

import (
	"io/ioutil"
	"os/exec"
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