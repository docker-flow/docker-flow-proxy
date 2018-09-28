package main

import (
	"log"
	"os"
	"strings"

	"github.com/docker-flow/docker-flow-proxy/logging"
)

func main() {
	log.SetOutput(os.Stdout)
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		go logging.StartLogging()
	}

	// TODO: Change to serverImpl.Execute
	newArgs().parse()
}
