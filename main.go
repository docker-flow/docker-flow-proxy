package main

import (
	"./logging"
	"strings"
	"os"
)

func main() {
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		go logging.StartLogging()
	}
	NewArgs().Parse()
}
