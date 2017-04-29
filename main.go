package main

import (
	"./logging"
	"os"
	"strings"
)

func main() {
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		go logging.StartLogging()
	}
	// TODO: Change to serverImpl.Execute
	NewArgs().Parse()
}
