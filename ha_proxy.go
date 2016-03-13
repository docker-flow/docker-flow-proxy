package main

import (
	"os"
	"fmt"
)

var ConsulDir = "/cfg/tmpl"
var ConsulTemplatePath = "/cfg/tmpl/service-formatted.ctmpl"
var ConfigsDir = "/cfg/tmpl"

type Proxy interface {
	RunCmd(extraArgs []string) error
}

type HaProxy struct { }

func (p HaProxy) RunCmd(extraArgs []string) error {
	args := []string{
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	args = append(args, extraArgs...)
	cmd := execHaCmd("haproxy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Command %v\n%v\n", cmd, err)
	}
	return nil
}
