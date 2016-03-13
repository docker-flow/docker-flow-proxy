package main

import (
	"github.com/jessevdk/go-flags"
	"os"
	"fmt"
)

type Args struct {}

type ArgsRun struct {}

var argsRun ArgsRun

func (x *ArgsRun) Execute(args []string) error {
	// TODO: Test
	// TODO: Call Run
	return nil
}

type ArgsReconfigure struct {
	ServiceName 	string	`short:"s" long:"service-name" required:"true" description:"The name of the service that should be reconfigured (e.g. my-service)."`
	ServicePath 	string	`short:"p" long:"service-path" required:"true" description:"Path that should be configured in the proxy (e.g. /api/v1/my-service)."`
	ConsulAddress	string	`short:"a" long:"consul-address" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
}

var argsReconfigure ArgsReconfigure

func (x *ArgsReconfigure) Execute(args []string) error {
	// TODO: Test
	// TODO: Call Reconfigure
	return nil
}

func (a Args) Parse() error {
	parser := flags.NewParser(nil, flags.Default)
	parser.AddCommand("run", "Runs the proxy", "Runs the proxy", &argsRun)
	parser.AddCommand("reconfigure", "Reconfigures the proxy", "Reconfigures the proxy using information stored in Consul", &argsReconfigure)
	if _, err := parser.ParseArgs(os.Args[1:]); err != nil {
		return fmt.Errorf("Could not parse command line arguments\n%v", err)
	}
	return nil
}