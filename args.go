package main

import (
	"github.com/jessevdk/go-flags"
	"os"
)

type Args struct{}

var NewArgs = func() Args {
	return Args{}
}

func (a Args) Parse() error {
	parser := flags.NewParser(nil, flags.Default)
	parser.AddCommand("server", "Runs the server", "Runs the server", &serverImpl)
	parser.ParseArgs(os.Args[1:])
	return nil
}
