package proxy

import (
	"os/exec"
	"io/ioutil"
	"log"
)

var cmdRunHa = func(cmd *exec.Cmd) error {
	return cmd.Run()
}
var readConfigsFile = ioutil.ReadFile
var writeFile = ioutil.WriteFile
var readFile = ioutil.ReadFile
var logPrintf = log.Printf
var readPidFile = ioutil.ReadFile
var readConfigsDir = ioutil.ReadDir