// +build !integration

package main

import (
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"os/exec"
	"testing"
)

type RemoveTestSuite struct {
	suite.Suite
	remove        Remove
	ServiceName   string
	ConfigsPath   string
	TemplatesPath string
}

func (s *RemoveTestSuite) SetupTest() {
	s.ServiceName = "myService"
	s.TemplatesPath = "/path/to/templates"
	s.ConfigsPath = "/path/to/configs"
	osRemove = func(name string) error {
		return nil
	}
	s.remove = Remove{
		ServiceName:   s.ServiceName,
		ConfigsPath:   s.ConfigsPath,
		TemplatesPath: s.TemplatesPath,
	}
}

// Execute

func (s RemoveTestSuite) Test_Execute_RemovesConfigurationFile() {
	var actual string
	expected := fmt.Sprintf("%s/%s.cfg", s.TemplatesPath, s.ServiceName)
	osRemove = func(name string) error {
		actual = name
		return nil
	}

	s.remove.Execute([]string{})

	s.Equal(expected, actual)
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenFailure() {
	osRemove = func(name string) error {
		return fmt.Errorf("The file could not be removed")
	}

	err := s.remove.Execute([]string{})

	s.Error(err)
}

func (s RemoveTestSuite) Test_Execute_Invokes_HaProxyCreateConfigFromTemplates() {
	proxyOrig := proxy
	defer func() {
		proxy = proxyOrig
	}()
	mockObj := getProxyMock("")
	proxy = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertCalled(s.T(), "CreateConfigFromTemplates", s.TemplatesPath, s.ConfigsPath)
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenHaProxyCreateConfigFromTemplatesFails() {
	proxyOrig := proxy
	defer func() {
		proxy = proxyOrig
	}()
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxy = mockObj

	err := s.remove.Execute([]string{})

	s.Error(err)
}

func (s RemoveTestSuite) Test_Execute_Invokes_HaProxyReload() {
	proxyOrig := proxy
	defer func() {
		proxy = proxyOrig
	}()
	mockObj := getProxyMock("")
	proxy = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertCalled(s.T(), "Reload")
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenHaProxyReloadFails() {
	proxyOrig := proxy
	defer func() {
		proxy = proxyOrig
	}()
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxy = mockObj

	err := s.remove.Execute([]string{})

	s.Error(err)
}

// Suite

func TestRemoveTestSuite(t *testing.T) {
	suite.Run(t, new(RemoveTestSuite))
}

// Mock

func (s RemoveTestSuite) mockConsulExecCmd() *[]string {
	var actualCommand []string
	cmdRunConsul = func(cmd *exec.Cmd) error {
		actualCommand = cmd.Args
		return nil
	}
	return &actualCommand
}

type RemoveMock struct {
	mock.Mock
}

func (m *RemoveMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func getRemoveMock(skipMethod string) *RemoveMock {
	mockObj := new(RemoveMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	return mockObj
}
