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
	NewArgs().Parse()
}
