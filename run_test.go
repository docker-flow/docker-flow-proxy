// +build !integration

package main

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RunTestSuite struct {
	suite.Suite
}

func (s *RunTestSuite) SetupTest() {
	logPrintf = func(format string, v ...interface{}) {}
}

// NewRun

func (s RunTestSuite) Test_NewRun_ReturnsNewStruct() {
	s.NotNil(NewRun())
}

// Suite

func TestRunUnitTestSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}

// Mock

type RunMock struct {
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
