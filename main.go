package main

import (
	"./logging"
	"log"
	"os"
	"strings"
	//	"sync"
	//	"github.com/hashicorp/go-reap"
)

func main() {
	log.SetOutput(os.Stdout)
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		go logging.StartLogging()
	}

	// TODO: Change to serverImpl.Execute
	newArgs().parse()
}
