package actions

import (
	"../proxy"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ReloadTestSuite struct {
	suite.Suite
}

func TestReloadUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ReloadTestSuite))
}

func (s *ReloadTestSuite) SetupTest() {
}

// Execute

func (s *ReloadTestSuite) Test_Execute_Invokes_HaProxyReload() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("")
	proxy.Instance = mockObj
	reload := Reload{}

	reload.Execute(false, "")

	mockObj.AssertCalled(s.T(), "Reload")
}

func (s *ReloadTestSuite) Test_Execute_ReturnsError_WhenHaProxyReloadFails() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("Reload")
	mockObj.On("Reload").Return(fmt.Errorf("This is an error"))
	proxy.Instance = mockObj
	reload := Reload{}

	err := reload.Execute(false, "")

	s.Error(err)
}

func (s *ReloadTestSuite) Test_Execute_InvokesCreateConfigFromTemplates_WhenRecreateIsTrue() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("")
	proxy.Instance = mockObj
	reload := Reload{}

	reload.Execute(true, "")

	mockObj.AssertCalled(s.T(), "CreateConfigFromTemplates")
}

func (s *ReloadTestSuite) Test_Execute_ReturnsError_WhenCreateConfigFromTemplatesFails() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("CreateConfigFromTemplates")
	mockObj.On("CreateConfigFromTemplates").Return(fmt.Errorf("This is an error"))
	proxy.Instance = mockObj
	reload := Reload{}

	err := reload.Execute(true, "")

	s.Error(err)
}

func (s *ReloadTestSuite) Test_Execute_InvokesReloadAllServices_WhenFromListenerIsTrue() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock
	reload := Reload{}
	newReconfigureOrig := NewReconfigure
	defer func() { NewReconfigure = newReconfigureOrig }()
	reconfigureMock := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData proxy.Service, mode string) Reconfigurable {
		return reconfigureMock
	}

	reload.Execute(false, "listener-addr")

	reconfigureMock.AssertCalled(s.T(), "ReloadAllServices", []string{}, "", "", "listener-addr")
}

func (s *ReloadTestSuite) Test_Execute_ReturnsError_WhenReloadAllServicesFails() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("")
	proxy.Instance = mockObj
	reload := Reload{}
	newReconfigureOrig := NewReconfigure
	defer func() { NewReconfigure = newReconfigureOrig }()
	reconfigureMock := getReconfigureMock("ReloadAllServices")
	reconfigureMock.On("ReloadAllServices", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("This is an error"))
	NewReconfigure = func(baseData BaseReconfigure, serviceData proxy.Service, mode string) Reconfigurable {
		return reconfigureMock
	}

	err := reload.Execute(false, "listener-addr")

	s.Error(err)
}

func (s *ReloadTestSuite) Test_Execute_DoesNotInvokeCreateConfigFromTemplates_WhenFromListenerIsTrue() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock
	reload := Reload{}
	newReconfigureOrig := NewReconfigure
	defer func() { NewReconfigure = newReconfigureOrig }()
	reconfigureMock := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData proxy.Service, mode string) Reconfigurable {
		return reconfigureMock
	}

	reload.Execute(true, "listener-addr")

	proxyMock.AssertNotCalled(s.T(), "CreateConfigFromTemplates")
}

func (s *ReloadTestSuite) Test_Execute_DoesNotInvokeReload_WhenFromListenerIsTrue() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	proxyMock := getProxyMock("")
	proxy.Instance = proxyMock
	reload := Reload{}
	newReconfigureOrig := NewReconfigure
	defer func() { NewReconfigure = newReconfigureOrig }()
	reconfigureMock := getReconfigureMock("")
	NewReconfigure = func(baseData BaseReconfigure, serviceData proxy.Service, mode string) Reconfigurable {
		return reconfigureMock
	}

	reload.Execute(true, "listener-addr")

	proxyMock.AssertNotCalled(s.T(), "Reload")
}

// NewReload

func (s *ReloadTestSuite) Test_NewReload_ReturnsNewInstance() {
	r := NewReload()

	s.NotNil(r)
}
