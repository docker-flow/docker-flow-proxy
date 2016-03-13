package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os/exec"
)

// Setup

type HaProxyTestSuite struct {
	suite.Suite
}

func (s *HaProxyTestSuite) SetupTest() {}

// Suite

func TestHaProxyTestSuite(t *testing.T) {
	suite.Run(t, new(HaProxyTestSuite))
}

// Helper

func (s HaProxyTestSuite) mockHaExecCmd() *[]string {
	var actualCommand []string
	execHaCmd = func(name string, arg ...string) *exec.Cmd {
		actualCommand = append([]string{name}, arg...)
		return &exec.Cmd{}
	}
	return &actualCommand
}

func (s HaProxyTestSuite) mockConsulExecCmd() *[]string {
	var actualCommand []string
	execConsulCmd = func(name string, arg ...string) *exec.Cmd {
		actualCommand = append([]string{name}, arg...)
		return &exec.Cmd{}
	}
	return &actualCommand
}

func (s HaProxyTestSuite) mockReadFileForConfigs() (*[]string, *string) {
	var files = []string{}
	var content = ""
	counter := 0
	readFile = func(filename string) ([]byte, error) {
		files = append(files, filename)
		content = string(string(counter))
		counter += 1
		return []byte(string(counter)), nil
	}
	return &files, &content
}
