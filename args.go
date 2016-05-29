package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

type Args struct{}

var NewArgs = func() Args {
	return Args{}
}

func (a Args) Parse() error {
	parser := flags.NewParser(nil, flags.Default)
	parser.AddCommand("server", "Runs the server", "Runs the server", &server)
	parser.AddCommand("run", "Runs the proxy", "Runs the proxy", &run)
	parser.AddCommand("reconfigure", "Reconfigures the proxy", "Reconfigures the proxy using information stored in Consul", &reconfigure)
	parser.AddCommand("remove", "Removes a service from the proxy", "Removes a service from the proxy", &remove)
	if _, err := parser.ParseArgs(os.Args[1:]); err != nil {
		return fmt.Errorf("Could not parse command line arguments\n%s", err.Error())
	}
	return nil
}
