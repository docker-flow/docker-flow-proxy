package actions

import (
	"../proxy"
	"fmt"
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

	reload.Execute()

	mockObj.AssertCalled(s.T(), "Reload")
}

func (s *ReloadTestSuite) Test_Execute_ReturnsError_WhenHaProxyReloadFails() {
	proxyOrig := proxy.Instance
	defer func() { proxy.Instance = proxyOrig }()
	mockObj := getProxyMock("Reload")
	mockObj.On("Reload").Return(fmt.Errorf("This is an error"))
	proxy.Instance = mockObj
	reload := Reload{}

	err := reload.Execute()

	s.Error(err)
}

// NewReload

func (s *ReloadTestSuite) Test_NewReload_ReturnsNewInstance() {
	r := NewReload()

	s.NotNil(r)
}
