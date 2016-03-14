package main

import (
//	"os"
//	"fmt"
	"os"
	"fmt"
	"os/exec"
)

const ServiceTemplateFilename = "service-formatted.ctmpl"

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
	cmd := exec.Command("haproxy", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmdRunHa(cmd); err != nil {
		return fmt.Errorf("Command %v\n%v\n", cmd, err)
	}
	return nil
}
