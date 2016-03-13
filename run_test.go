package main

import (
"github.com/stretchr/testify/suite"
"testing"
)

type RunTestSuite struct {
	suite.Suite
}

func (s *RunTestSuite) SetupTest() {
}

// Execute

func (s RunTestSuite) Test_Execute_ExecutesCommand() {
	actual := HaProxyTestSuite{}.mockHaExecCmd()
	expected := []string{
		"haproxy",
		"-f",
		"/cfg/haproxy.cfg",
		"-D",
		"-p",
		"/var/run/haproxy.pid",
	}
	Run{}.Execute([]string{})

	s.Equal(expected, *actual)
}

// Suite

func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}