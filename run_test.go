package main

import (
"github.com/stretchr/testify/suite"
"testing"
	"github.com/stretchr/testify/mock"
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
	NewRun().Execute([]string{})

	s.Equal(expected, *actual)
}

// NewRun

func (s RunTestSuite) Test_NewRun_ReturnsNewStruct() {
	s.NotNil(NewRun())
}

// Suite

func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}

// Mock

type RunMock struct{
	mock.Mock
}

func (m *RunMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func getRunMock(skipMethod string) *ReconfigureMock {
	mockObj := new(ReconfigureMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	return mockObj
}
