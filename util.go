package main

import (
	"io/ioutil"
	"os/exec"
)

var readFile = ioutil.ReadFile
var readDir = ioutil.ReadDir
var writeFile = ioutil.WriteFile
var execConsulCmd = exec.Command
var execHaCmd = exec.Command