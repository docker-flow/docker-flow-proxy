package main

import (
	"io/ioutil"
	"os/exec"
)

var readFile = ioutil.ReadFile
var readDir = ioutil.ReadDir
var writeFile = ioutil.WriteFile
var execCmd = exec.Command
