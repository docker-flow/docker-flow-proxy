package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unicode"
)

var haProxyCmd = "haproxy"

var cmdRunHa = func(args []string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(haProxyCmd, args...)

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Start()

	go func() {
		io.Copy(stdout, stdoutIn)
	}()

	go func() {
		io.Copy(stderr, stderrIn)
	}()

	err := cmd.Wait()

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	combinedOut := fmt.Sprintf("\nstdout:\n%s\nstderr:\n%s\n", outStr, errStr)

	if exitError, ok := err.(*exec.ExitError); ok {
		waitStatus := exitError.Sys().(syscall.WaitStatus)
		fmt.Printf("Exit Status: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		return fmt.Errorf(combinedOut)
	}

	if errStr != "" {
		fmt.Println("The configuration file is valid, but there still may be a misconfiguration",
			"somewhere that will give unexpected results, please verify:", combinedOut)
	}

	return nil
}

var cmdValidateHa = func(args []string) error {
	return cmdRunHa(args)
}
var readConfigsFile = ioutil.ReadFile
var readSecretsFile = ioutil.ReadFile
var writeFile = ioutil.WriteFile
var lookupHost = net.LookupHost

// ReadFile overwrites ioutil.ReadFile so that it can be mocked from other packages
var ReadFile = ioutil.ReadFile
var readDir = ioutil.ReadDir
var readFile = ioutil.ReadFile
var logPrintf = log.Printf
var readPidFile = ioutil.ReadFile
var readConfigsDir = ioutil.ReadDir
var haSocketOn = func(address string) bool {
	conn, err := net.Dial("unix", address)
	defer conn.Close()
	return err == nil
}
var getSecretOrEnvVarSplit = func(key, defaultValue string) string {
	value := getSecretOrEnvVar(key, defaultValue)
	if len(value) > 0 {
		value = strings.Replace(value, ",", "\n    ", -1)
	}
	return value
}
var getSecretOrEnvVar = func(key, defaultValue string) string {
	path := fmt.Sprintf("/run/secrets/dfp_%s", strings.ToLower(key))
	if content, err := readSecretsFile(path); err == nil {
		return strings.TrimRight(string(content[:]), "\n")
	}
	if len(os.Getenv(key)) > 0 {
		return os.Getenv(key)
	}
	return defaultValue
}
var lowerFirst = func(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(append([]rune{unicode.ToLower([]rune(s)[0])}, []rune(s)[1:]...))
}

// IsValidReconf validates whether reconfigure data is valid
func IsValidReconf(service *Service) (statusCode int, msg string) {
	reqMode := "http"
	if len(service.ServiceName) == 0 {
		return http.StatusBadRequest, "serviceName parameter is mandatory."
	} else if len(service.ServiceDest[0].ReqMode) > 0 {
		reqMode = service.ServiceDest[0].ReqMode
	}
	hasPath := len(service.ServiceDest[0].ServicePath) > 0
	hasSrcPort := service.ServiceDest[0].SrcPort > 0
	hasPort := len(service.ServiceDest[0].Port) > 0
	hasDomain := len(service.ServiceDest[0].ServiceDomain) > 0
	//	hasDomain := len(service.ServiceDomain) > 0
	if strings.EqualFold(reqMode, "http") {
		if !hasPath && !hasDomain {
			return http.StatusConflict, "When using reqMode http, servicePath or serviceDomain are mandatory"
		}
	} else if !hasSrcPort || !hasPort {
		return http.StatusBadRequest, "When NOT using reqMode http (e.g. tcp), srcPort and port parameters are mandatory."
	}
	return http.StatusOK, ""
}

func replaceNonAlphabetAndNumbers(value []string) string {
	reg, _ := regexp.Compile("[^A-Za-z0-9]+")
	return reg.ReplaceAllString(strings.Join(value, "_"), "_")
}

var mu = &sync.Mutex{}
