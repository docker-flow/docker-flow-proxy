// +build !integration

package main

import (
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RemoveTestSuite struct {
	suite.Suite
	remove        Remove
	ServiceName   string
	ConfigsPath   string
	TemplatesPath string
	ConsulAddress string
	InstanceName  string
}

func (s *RemoveTestSuite) SetupTest() {
	s.ServiceName = "myService"
	s.TemplatesPath = "/path/to/templates"
	s.ConfigsPath = "/path/to/configs"
	s.ConsulAddress = "http://consul.io"
	s.InstanceName = "my-proxy-instance"
	osRemove = func(name string) error {
		return nil
	}
	s.remove = Remove{
		ServiceName:     s.ServiceName,
		ConfigsPath:     s.ConfigsPath,
		TemplatesPath:   s.TemplatesPath,
		ConsulAddresses: []string{s.ConsulAddress},
		InstanceName:    s.InstanceName,
	}
}

// Execute

func (s RemoveTestSuite) Test_Execute_RemovesConfigurationFile() {
	var actual []string
	expected := []string{
		fmt.Sprintf("%s/%s-fe.cfg", s.TemplatesPath, s.ServiceName),
		fmt.Sprintf("%s/%s-be.cfg", s.TemplatesPath, s.ServiceName),
	}
	osRemove = func(name string) error {
		actual = append(actual, name)
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
	defer func() { proxy = proxyOrig }()
	mockObj := getProxyMock("")
	proxy = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertCalled(s.T(), "CreateConfigFromTemplates", s.TemplatesPath, s.ConfigsPath)
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenHaProxyCreateConfigFromTemplatesFails() {
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxy = mockObj

	err := s.remove.Execute([]string{})

	s.Error(err)
}

func (s RemoveTestSuite) Test_Execute_Invokes_HaProxyReload() {
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	mockObj := getProxyMock("")
	proxy = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertCalled(s.T(), "Reload")
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenHaProxyReloadFails() {
	proxyOrig := proxy
	defer func() { proxy = proxyOrig }()
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates", mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	proxy = mockObj

	err := s.remove.Execute([]string{})

	s.Error(err)
}

func (s RemoveTestSuite) Test_Execute_InvokesRegistryDeleteService() {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertCalled(s.T(), "DeleteService", []string{s.ConsulAddress}, s.ServiceName, s.InstanceName)
}

func (s RemoveTestSuite) Test_Execute_DoesNotInvokeRegistryDeleteService_WhenModeIsService() {
	mockObj := getRegistrarableMock("")
	s.remove.Mode = "SerVIcE"
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "DeleteService", mock.Anything, mock.Anything, mock.Anything)
}

func (s RemoveTestSuite) Test_Execute_DoesNotInvokeRegistryDeleteService_WhenModeIsSwarm() {
	mockObj := getRegistrarableMock("")
	s.remove.Mode = "swARM"
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj

	s.remove.Execute([]string{})

	mockObj.AssertNotCalled(s.T(), "DeleteService", mock.Anything, mock.Anything, mock.Anything)
}

func (s RemoveTestSuite) Test_Execute_ReturnsError_WhenDeleteRequestToRegistryFails() {
	mockObj := getRegistrarableMock("DeleteService")
	mockObj.On("DeleteService", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error form Consul"))
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj

	err := s.remove.Execute([]string{})

	s.Error(err)
}

// Suite

func TestRemoveUnitTestSuite(t *testing.T) {
	mockObj := getRegistrarableMock("")
	registryInstanceOrig := registryInstance
	defer func() { registryInstance = registryInstanceOrig }()
	registryInstance = mockObj
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, new(RemoveTestSuite))
}

// Mock

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
