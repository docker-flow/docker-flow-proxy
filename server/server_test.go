package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/docker-flow/docker-flow-proxy/actions"
	"github.com/docker-flow/docker-flow-proxy/proxy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	BaseUrl     string
	serviceName string
	suite.Suite
}

func (s *ServerTestSuite) SetupTest() {
}

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	s.serviceName = "my-fancy-service"

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	os.Setenv("SKIP_ADDRESS_VALIDATION", "false")
	os.Setenv("SEPARATOR", ",")

	suite.Run(t, s)
}

// Test1Handler

func (s *ServerTestSuite) Test_Test1Handler_ReturnsStatus200() {
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/test", nil)

	srv := serve{}
	srv.Test1Handler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

// Test2Handler

func (s *ServerTestSuite) Test_Test2Handler_ReturnsStatus200() {
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", "/v2/test", nil)

		srv := serve{}
		srv.Test2Handler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

// PingHandler

func (s *ServerTestSuite) Test_PingHandler_ReturnsStatus200() {
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/ping", nil)

	srv := serve{}
	srv.PingHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

// ReconfigureHandler

func (s *ServerTestSuite) Test_ReconfigureHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reconfigure?serviceName=my-service", nil)

	srv := serve{}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesErrorHeader_WhenReconfigureDistributeIsTrueAndError() {
	serve := serve{}
	serve.port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo&port=1234"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := sendDistributeRequests
	defer func() { sendDistributeRequests = sendDistributeRequestsOrig }()
	sendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, fmt.Errorf("This is an error")
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesStatusOK_WhenReconfigureDistributeIsTrue() {
	serve := serve{}
	serve.port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo&port=1234"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := sendDistributeRequests
	defer func() { sendDistributeRequests = sendDistributeRequestsOrig }()
	sendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, nil
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenServiceNameQueryIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus409_WhenModeIsHttpAndServicePathAndServiceDomainAreEmpty() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 409)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus200_WhenReqModeIsTcp() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				return nil
			},
		}
	}

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndSrcPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenPortIsNotPresent() {
	url := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&serviceColor=orange&servicePath=/demo&serviceDomain=my-domain.com"
	req, _ := http.NewRequest("GET", url, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesReconfigureExecute() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath.1=/demo&port.1=1234"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				invoked = true
				return nil
			},
		}
	}

	srv := serve{}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_DoesNotInvokeReconfigureExecute_WhenDistributeIsTrue() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				invoked = true
				return nil
			},
		}
	}

	srv := serve{}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.False(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus500_WhenReconfigureExecuteFails() {
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				return fmt.Errorf("This is an error")
			},
		}
	}
	rw := getResponseWriterMock()
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234"
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCert_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{
		cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-service", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCertWithDomainName_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceDomain=my-domain.com&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{
		cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-domain.com", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

// ReloadHandler

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsStatus200() {
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

		srv := serve{}
		srv.ReloadHandler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReload() {
	invoked := false
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			invoked = true
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

	srv := serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithRecreateParam() {
	actualRecreate := false
	clusterConfigCalled := false
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			actualRecreate = recreate
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			clusterConfigCalled = true
			return nil
		},
	})()

	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?recreate=true", nil)

	srv := serve{}
	srv.listenerAddresses = []string{"my-listener"}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(actualRecreate)
	s.False(clusterConfigCalled)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithFromListenerParam() {
	actualListenerAddr := ""
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			actualListenerAddr = listenerAddr
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?fromListener=true", nil)

	srv := serve{}
	srv.listenerAddresses = []string{"my-listener"}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal(srv.listenerAddresses[0], actualListenerAddr)
}
func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithFromListenerParam_MultipleListeners() {
	rw := getResponseWriterMock()
	actualListenerAddrs := []string{}
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			actualListenerAddrs = append(
				actualListenerAddrs, listenerAddr)
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?fromListener=true", nil)

	srv := serve{}
	srv.listenerAddresses = []string{"dfsl1", "dfsl2"}
	srv.ReloadHandler(rw, req)

	for _, addr := range srv.listenerAddresses {
		s.Contains(actualListenerAddrs, addr)
	}

	rw.AssertCalled(s.T(), "WriteHeader", http.StatusOK)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithFromListenerParam_MultipleListeners_OneErrors() {
	rw := getResponseWriterMock()
	actualListenerAddrs := []string{}
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			actualListenerAddrs = append(
				actualListenerAddrs, listenerAddr)
			if listenerAddr == "dfsl2" {
				return errors.New("bad")
			}
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?fromListener=true", nil)

	srv := serve{}
	srv.listenerAddresses = []string{"dfsl1", "dfsl2"}
	srv.ReloadHandler(rw, req)

	for _, addr := range srv.listenerAddresses {
		s.Contains(actualListenerAddrs, addr)
	}
	rw.AssertCalled(s.T(), "WriteHeader", http.StatusInternalServerError)
}

func (s *ServerTestSuite) Test_ReloadHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()

	srv := serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status: "OK",
	})
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.ReloadHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

// Remove Handler

func (s *ServerTestSuite) Test_RemoveHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)

	srv := serve{}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: "my-service",
	})
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsStatus400_WhenServiceNameIsNotPresent() {
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_RemoveHandler_InvokesRemoveExecute() {
	mockObj := getRemoveMock("")
	aclName := "my-acl"
	var actual actions.Remove
	expected := actions.Remove{
		ServiceName:   s.serviceName,
		TemplatesPath: "",
		ConfigsPath:   "",
		InstanceName:  "proxy-test-instance",
		AclName:       aclName,
	}
	actions.NewRemove = func(
		serviceName, aclName, configsPath, templatesPath string,
		instanceName string,
	) actions.Removable {
		actual = actions.Remove{
			ServiceName:   serviceName,
			AclName:       aclName,
			TemplatesPath: templatesPath,
			ConfigsPath:   configsPath,
			InstanceName:  instanceName,
		}
		return mockObj
	}
	url := fmt.Sprintf("/v1/docker-flow-proxy/remove?serviceName=%s&aclName=%s", s.serviceName, aclName)
	req, _ := http.NewRequest("GET", url, nil)

	srv := serve{
		serviceName: expected.InstanceName,
	}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal(expected, actual)
	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

func (s *ServerTestSuite) Test_RemoveHandler_WritesErrorHeader_WhenRemoveDistributeIsTrueAndItFails() {
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	serve := serve{}
	serve.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 500)
}

// GetServiceFromUrl

func (s *ServerTestSuite) Test_GetServiceFromUrl_ReturnsProxyService() {
	expected := proxy.Service{
		AclName:               "aclName",
		AddReqHeader:          []string{"add-header-1", "add-header-2"},
		AddResHeader:          []string{"add-header-1", "add-header-2"},
		CheckResolvers:        true,
		CompressionAlgo:       "compressionAlgo",
		CompressionType:       "compressionType",
		ConnectionMode:        "my-connection-mode",
		DelReqHeader:          []string{"add-header-1", "add-header-2"},
		DelResHeader:          []string{"add-header-1", "add-header-2"},
		DiscoveryType:         "DNS",
		Distribute:            true,
		ProxyInstanceName:     "docker-flow",
		RedirectWhenHttpProto: true,
		Replicas:              83,
		ReqPathReplace:        "reqPathReplace",
		ReqPathSearch:         "reqPathSearch",
		ServiceCert:           "serviceCert",
		ServiceDest: []proxy.ServiceDest{{
			AllowedMethods:     []string{"GET", "DELETE"},
			BalanceGroup:       "leastconn",
			CheckTCP:           true,
			Clitcpka:           true,
			DeniedMethods:      []string{"PUT", "POST"},
			HttpsOnly:          true,
			HttpsPort:          1234,
			HttpsRedirectCode:  "302",
			OutboundHostname:   "outboundHostname",
			PathType:           "pathType",
			Port:               "1234",
			RedirectFromDomain: []string{"sub.domain1", "sub.domain2"},
			ReqMode:            "reqMode",
			ReqPathSearchReplaceFormatted: []string{"reqPathSearch,reqPathReplace"},
			ServiceDomain:                 []string{"domain1", "domain2"},
			ServiceHeader:                 map[string]string{"X-Version": "3", "name": "Viktor"},
			ServicePath:                   []string{"/"},
			ServicePathExclude:            []string{"/excluded-path"},
			SrcPort:                       8080,
			SrcHttpsPort:                  4443,
			SslVerifyNone:                 true,
			TimeoutServer:                 "timeoutServer",
			TimeoutClient:                 "timeoutClient",
			TimeoutTunnel:                 "timeoutTunnel",
		}},
		ServiceDomainAlgo: "hdr_dom",
		ServiceName:       "serviceName",
		SetReqHeader:      []string{"set-header-1", "set-header-2"},
		SetResHeader:      []string{"set-header-1", "set-header-2"},
		TemplateBePath:    "templateBePath",
		TemplateFePath:    "templateFePath",
		Users: []proxy.User{{Username: "user1", Password: "pass1", PassEncrypted: true},
			{Username: "user2", Password: "pass2", PassEncrypted: true}},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&users=%s&usersPassEncrypted=%t&aclName=%s&balanceGroup=%s&checkTcp=%t&clitcpka=%t&serviceCert=%s&outboundHostname=%s&pathType=%s&proxyInstanceName=%s&reqPathSearch=%s&reqPathReplace=%s&templateFePath=%s&templateBePath=%s&timeoutServer=%s&timeoutClient=%s&timeoutTunnel=%s&reqMode=%s&httpsOnly=%t&httpsRedirectCode=%s&isDefaultBackend=%t&redirectWhenHttpProto=%t&httpsPort=%d&srcPort=%d&srcHttpsPort=%d&serviceDomain=%s&redirectFromDomain=%s&distribute=%t&sslVerifyNone=%t&serviceDomainAlgo=%s&addReqHeader=%s&addResHeader=%s&setReqHeader=%s&setResHeader=%s&delReqHeader=%s&delResHeader=%s&servicePath=/&servicePathExclude=%s&port=1234&connectionMode=%s&serviceHeader=X-Version:3,name:Viktor&allowedMethods=GET,DELETE&deniedMethods=PUT,POST&compressionAlgo=%s&compressionType=%s&checkResolvers=%t&replicas=%d&discoveryType=%s",

		s.BaseUrl,
		expected.ServiceName,
		"user1:pass1,user2:pass2",
		true,
		expected.AclName,
		expected.ServiceDest[0].BalanceGroup,
		expected.ServiceDest[0].CheckTCP,
		expected.ServiceDest[0].Clitcpka,
		expected.ServiceCert,
		expected.ServiceDest[0].OutboundHostname,
		expected.ServiceDest[0].PathType,
		expected.ProxyInstanceName,
		expected.ReqPathSearch,
		expected.ReqPathReplace,
		expected.TemplateFePath,
		expected.TemplateBePath,
		expected.ServiceDest[0].TimeoutServer,
		expected.ServiceDest[0].TimeoutClient,
		expected.ServiceDest[0].TimeoutTunnel,
		expected.ServiceDest[0].ReqMode,
		expected.ServiceDest[0].HttpsOnly,
		expected.ServiceDest[0].HttpsRedirectCode,
		expected.IsDefaultBackend,
		expected.RedirectWhenHttpProto,
		expected.ServiceDest[0].HttpsPort,
		expected.ServiceDest[0].SrcPort,
		expected.ServiceDest[0].SrcHttpsPort,
		strings.Join(expected.ServiceDest[0].ServiceDomain, ","),
		strings.Join(expected.ServiceDest[0].RedirectFromDomain, ","),
		expected.Distribute,
		expected.ServiceDest[0].SslVerifyNone,
		expected.ServiceDomainAlgo,
		strings.Join(expected.AddReqHeader, ","),
		strings.Join(expected.AddResHeader, ","),
		strings.Join(expected.SetReqHeader, ","),
		strings.Join(expected.SetResHeader, ","),
		strings.Join(expected.DelReqHeader, ","),
		strings.Join(expected.DelResHeader, ","),
		strings.Join(expected.ServiceDest[0].ServicePathExclude, ","),
		expected.ConnectionMode,
		expected.CompressionAlgo,
		expected.CompressionType,
		expected.CheckResolvers,
		expected.Replicas,
		expected.DiscoveryType,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := serve{}
	actual := srv.GetServiceFromUrl(req)

	s.Equal(expected, *actual)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_SetsServiceDomainAlgoToHdrDom_WhenServiceDomainMatchAllIsSetToTrue() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path&serviceDomainMatchAll=true", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("hdr_dom(host)", actual.ServiceDomainAlgo)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_DefaultsReqModeToHttp() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("http", actual.ServiceDest[0].ReqMode)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_NoReplicas_IsGlobal() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("http", actual.ServiceDest[0].ReqMode)
	s.Equal(0, actual.Replicas)
	s.Equal(true, actual.IsGlobal)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_HasReplicas_NotGlobal() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path&replicas=2", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("http", actual.ServiceDest[0].ReqMode)
	s.Equal(2, actual.Replicas)
	s.Equal(false, actual.IsGlobal)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_ZeroReplicas_NotGlobal() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path&replicas=0", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("http", actual.ServiceDest[0].ReqMode)
	s.Equal(0, actual.Replicas)
	s.Equal(false, actual.IsGlobal)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_SetsServicePathToSlash_WhenDomainIsPresent() {
	expected := proxy.Service{
		ServiceName: "serviceName",
		IsGlobal:    true,
		ServiceDest: []proxy.ServiceDest{
			{
				AllowedMethods:     []string{},
				DeniedMethods:      []string{},
				Port:               "1234",
				RedirectFromDomain: []string{},
				ReqMode:            "http",
				ReqPathSearchReplaceFormatted: []string{},
				ServiceDomain:                 []string{"domain1", "domain2"},
				ServiceHeader:                 map[string]string{},
				ServicePath:                   []string{"/"},
				ServicePathExclude:            []string{},
				Index:                         0,
			},
		},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&serviceDomain=%s&port=%s",
		s.BaseUrl,
		expected.ServiceName,
		strings.Join(expected.ServiceDest[0].ServiceDomain, ","),
		expected.ServiceDest[0].Port,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal(expected, *actual)
}

// GetServicesFromEnvVars

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsServices() {
	service := proxy.Service{
		AclName:               "my-AclName",
		AddReqHeader:          []string{"add-header-1", "add-header-2"},
		AddResHeader:          []string{"add-header-1", "add-header-2"},
		CheckResolvers:        true,
		CompressionAlgo:       "compressionAlgo",
		ConnectionMode:        "my-connection-mode",
		DelReqHeader:          []string{"del-header-1", "del-header-2"},
		DelResHeader:          []string{"del-header-1", "del-header-2"},
		Distribute:            true,
		IsDefaultBackend:      true,
		RedirectWhenHttpProto: true,
		Replicas:              -1,
		ServiceCert:           "my-ServiceCert",
		ServiceDomainAlgo:     "hdr_dom",
		ServiceName:           "my-ServiceName",
		SetReqHeader:          []string{"set-header-1", "set-header-2"},
		SetResHeader:          []string{"set-header-1", "set-header-2"},
		TemplateBePath:        "my-TemplateBePath",
		TemplateFePath:        "my-TemplateFePath",
		ServiceDest: []proxy.ServiceDest{
			{
				HttpsOnly:                     true,
				HttpsPort:                     1234,
				HttpsRedirectCode:             "302",
				IgnoreAuthorization:           true,
				OutboundHostname:              "my-OutboundHostname",
				PathType:                      "my-PathType",
				Port:                          "1111",
				ReqPathSearchReplace:          "/something,/else:/this,/that",
				ReqPathSearchReplaceFormatted: []string{"/something,/else", "/this,/that"},
				ServiceDomain:                 []string{"my-domain-1.com", "my-domain-2.com"},
				ServicePath:                   []string{"my-path-11", "my-path-12"},
				SrcPort:                       1112,
				SrcHttpsPort:                  4443,
				ReqMode:                       "my-ReqMode",
				AllowedMethods:                []string{"GET", "POST"},
				DeniedMethods:                 []string{"OPTION", "TRACE"},
				RedirectFromDomain:            []string{"proxy.dockerflow.com", "dockerflow.com"},
				ServicePathExclude:            []string{"some-path", "some-path2"},
				VerifyClientSsl:               true,
				DenyHttp:                      true,
				SslVerifyNone:                 true,
				TimeoutServer:                 "my-TimeoutServer",
				TimeoutTunnel:                 "my-TimeoutTunnel",
			},
		},
	}
	os.Setenv("DFP_SERVICE_ACL_NAME", service.AclName)
	os.Setenv("DFP_SERVICE_ADD_REQ_HEADER", strings.Join(service.AddReqHeader, ","))
	os.Setenv("DFP_SERVICE_ADD_RES_HEADER", strings.Join(service.AddResHeader, ","))
	os.Setenv("DFP_SERVICE_ALLOWED_METHODS", strings.Join(service.ServiceDest[0].AllowedMethods, ","))
	os.Setenv("DFP_SERVICE_CHECK_RESOLVERS", strconv.FormatBool(service.CheckResolvers))
	os.Setenv("DFP_SERVICE_COMPRESSION_ALGO", service.CompressionAlgo)
	os.Setenv("DFP_SERVICE_COMPRESSION_TYPE", service.CompressionType)
	os.Setenv("DFP_SERVICE_CONNECTION_MODE", service.ConnectionMode)
	os.Setenv("DFP_SERVICE_DEL_REQ_HEADER", strings.Join(service.DelReqHeader, ","))
	os.Setenv("DFP_SERVICE_DEL_RES_HEADER", strings.Join(service.DelResHeader, ","))
	os.Setenv("DFP_SERVICE_DENIED_METHODS", strings.Join(service.ServiceDest[0].DeniedMethods, ","))
	os.Setenv("DFP_SERVICE_DENY_HTTP", strconv.FormatBool(service.ServiceDest[0].DenyHttp))
	os.Setenv("DFP_SERVICE_DISTRIBUTE", strconv.FormatBool(service.Distribute))
	os.Setenv("DFP_SERVICE_HTTPS_ONLY", strconv.FormatBool(service.ServiceDest[0].HttpsOnly))
	os.Setenv("DFP_SERVICE_HTTPS_REDIRECT_CODE", service.ServiceDest[0].HttpsRedirectCode)
	os.Setenv("DFP_SERVICE_HTTPS_PORT", strconv.Itoa(service.ServiceDest[0].HttpsPort))
	os.Setenv("DFP_SERVICE_IGNORE_AUTHORIZATION", strconv.FormatBool(service.ServiceDest[0].IgnoreAuthorization))
	os.Setenv("DFP_SERVICE_IS_DEFAULT_BACKEND", strconv.FormatBool(service.IsDefaultBackend))
	os.Setenv("DFP_SERVICE_OUTBOUND_HOSTNAME", service.ServiceDest[0].OutboundHostname)
	os.Setenv("DFP_SERVICE_PATH_TYPE", service.ServiceDest[0].PathType)
	os.Setenv("DFP_SERVICE_REDIRECT_FROM_DOMAIN", strings.Join(service.ServiceDest[0].RedirectFromDomain, ","))
	os.Setenv("DFP_SERVICE_REDIRECT_WHEN_HTTP_PROTO", strconv.FormatBool(service.RedirectWhenHttpProto))
	os.Setenv("DFP_SERVICE_REQ_MODE", service.ServiceDest[0].ReqMode)
	os.Setenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE", service.ServiceDest[0].ReqPathSearchReplace)
	os.Setenv("DFP_SERVICE_SERVICE_CERT", service.ServiceCert)
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN", strings.Join(service.ServiceDest[0].ServiceDomain, ","))
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN_ALGO", service.ServiceDomainAlgo)
	os.Setenv("DFP_SERVICE_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_SERVICE_PATH_EXCLUDE", strings.Join(service.ServiceDest[0].ServicePathExclude, ","))
	os.Setenv("DFP_SERVICE_SSL_VERIFY_NONE", strconv.FormatBool(service.ServiceDest[0].SslVerifyNone))
	os.Setenv("DFP_SERVICE_TEMPLATE_BE_PATH", service.TemplateBePath)
	os.Setenv("DFP_SERVICE_TEMPLATE_FE_PATH", service.TemplateFePath)
	os.Setenv("DFP_SERVICE_TIMEOUT_SERVER", service.ServiceDest[0].TimeoutServer)
	os.Setenv("DFP_SERVICE_TIMEOUT_TUNNEL", service.ServiceDest[0].TimeoutTunnel)
	os.Setenv("DFP_SERVICE_PORT", service.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_SERVICE_PATH", strings.Join(service.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SET_REQ_HEADER", strings.Join(service.SetReqHeader, ","))
	os.Setenv("DFP_SERVICE_SET_RES_HEADER", strings.Join(service.SetResHeader, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT", strconv.Itoa(service.ServiceDest[0].SrcPort))
	os.Setenv("DFP_SERVICE_SRC_HTTPS_PORT", strconv.Itoa(service.ServiceDest[0].SrcHttpsPort))
	os.Setenv("DFP_SERVICE_VERIFY_CLIENT_SSL", strconv.FormatBool(service.ServiceDest[0].VerifyClientSsl))
	// os.Setenv("DFP_SERVICE_SSL_VERIFY_NONE", strconv.FormatBool(service.ServiceDest[0].VerifyClientSsl))

	defer func() {
		os.Unsetenv("DFP_SERVICE_ACL_NAME")
		os.Unsetenv("DFP_SERVICE_ADD_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_ADD_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_ALLOWED_METHODS")
		os.Unsetenv("DFP_SERVICE_CHECK_RESOLVERS")
		os.Unsetenv("DFP_SERVICE_COMPRESSION_ALGO")
		os.Unsetenv("DFP_SERVICE_COMPRESSION_TYPE")
		os.Unsetenv("DFP_SERVICE_CONNECTION_MODE")
		os.Unsetenv("DFP_SERVICE_DEL_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_DEL_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_DENIED_METHODS")
		os.Unsetenv("DFP_SERVICE_DENY_HTTP")
		os.Unsetenv("DFP_SERVICE_DISTRIBUTE")
		os.Unsetenv("DFP_SERVICE_HTTPS_ONLY")
		os.Unsetenv("DFP_SERVICE_HTTPS_PORT")
		os.Unsetenv("DFP_SERVICE_HTTPS_REDIRECT_CODE")
		os.Unsetenv("DFP_SERVICE_IGNORE_AUTHORIZATION")
		os.Unsetenv("DFP_SERVICE_IS_DEFAULT_BACKEND")
		os.Unsetenv("DFP_SERVICE_OUTBOUND_HOSTNAME")
		os.Unsetenv("DFP_SERVICE_PATH_TYPE")
		os.Unsetenv("DFP_SERVICE_PORT")
		os.Unsetenv("DFP_SERVICE_REDIRECT_FROM_DOMAIN")
		os.Unsetenv("DFP_SERVICE_REDIRECT_WHEN_HTTP_PROTO")
		os.Unsetenv("DFP_SERVICE_REQ_MODE")
		os.Unsetenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE")
		os.Unsetenv("DFP_SERVICE_SERVICE_CERT")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN_ALGO")
		os.Unsetenv("DFP_SERVICE_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_EXCLUDE")
		os.Unsetenv("DFP_SERVICE_SET_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_SET_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_SRC_PORT")
		os.Unsetenv("DFP_SERVICE_SRC_HTTPS_PORT")
		os.Unsetenv("DFP_SERVICE_SSL_VERIFY_NONE")
		os.Unsetenv("DFP_SERVICE_TEMPLATE_BE_PATH")
		os.Unsetenv("DFP_SERVICE_TEMPLATE_FE_PATH")
		os.Unsetenv("DFP_SERVICE_TIMEOUT_SERVER")
		os.Unsetenv("DFP_SERVICE_TIMEOUT_TUNNEL")
		os.Unsetenv("DFP_SERVICE_VERIFY_CLIENT_SSL")
		os.Unsetenv("DFP_SERVICE_X_FORWARDED_PROTO")
	}()
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_SetsServiceDomainAlgoToHdrDom_WhenServiceDomainMatchAllIsTrue() {
	service := proxy.Service{
		ServiceName: "my-ServiceName",
		Replicas:    -1,
		ServiceDest: []proxy.ServiceDest{
			{
				ReqMode: "http",
				ReqPathSearchReplaceFormatted: []string{},
				ServiceDomain:                 []string{"my-domain-1.com", "my-domain-2.com"},
				ServicePath:                   []string{"my-path-11", "my-path-12"},
				AllowedMethods:                []string{},
				DeniedMethods:                 []string{},
				RedirectFromDomain:            []string{},
				ServicePathExclude:            []string{},
			},
		},
		ServiceDomainAlgo: "hdr_dom(host)",
	}
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN", strings.Join(service.ServiceDest[0].ServiceDomain, ","))
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN_MATCH_ALL", "true")
	os.Setenv("DFP_SERVICE_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_SERVICE_PATH", strings.Join(service.ServiceDest[0].ServicePath, ","))

	defer func() {
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN_MATCH_ALL")
		os.Unsetenv("DFP_SERVICE_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH")
	}()
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsServicesWithIndexedData() {
	expected := proxy.Service{
		ServiceName: "my-ServiceName",
		Replicas:    -1,
		ServiceDest: []proxy.ServiceDest{
			{
				Port:                          "1111",
				ReqPathSearchReplace:          "/this,/that",
				ReqPathSearchReplaceFormatted: []string{"/this,/that"},
				ServicePath:                   []string{"my-path-11", "my-path-12"},
				ServiceDomain:                 []string{"some-domain.com", "some-domain2.com"},
				AllowedMethods:                []string{},
				DeniedMethods:                 []string{},
				RedirectFromDomain:            []string{},
				ServicePathExclude:            []string{},
				SrcPort:                       1112,
				SrcHttpsPort:                  4443,
				HttpsOnly:                     true,
			}, {
				Port:                          "2221",
				ReqPathSearchReplace:          "/something,/else",
				ReqPathSearchReplaceFormatted: []string{"/something,/else"},
				ServicePath:                   []string{"my-path-21", "my-path-22"},
				ServiceDomain:                 []string{},
				SrcPort:                       2222,
				SrcHttpsPort:                  4444,
				HttpsOnly:                     false,
				OutboundHostname:              "my-outbound-domain.com",
				AllowedMethods:                []string{"GET", "POST"},
				DeniedMethods:                 []string{"OPTION", "TRACE"},
				RedirectFromDomain:            []string{"proxy.dockerflow.com", "dockerflow.com"},
				ServicePathExclude:            []string{"some-path", "some-path2"},
				DenyHttp:                      true,
				IgnoreAuthorization:           true,
			},
		},
	}
	os.Setenv("DFP_SERVICE_SERVICE_NAME", expected.ServiceName)
	os.Setenv("DFP_SERVICE_HTTPS_ONLY_1", "true")
	os.Setenv("DFP_SERVICE_PORT_1", expected.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE_1", expected.ServiceDest[0].ReqPathSearchReplace)
	os.Setenv("DFP_SERVICE_SERVICE_PATH_1", strings.Join(expected.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN_1", strings.Join(expected.ServiceDest[0].ServiceDomain, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT_1", strconv.Itoa(expected.ServiceDest[0].SrcPort))
	os.Setenv("DFP_SERVICE_SRC_HTTPS_PORT_1", strconv.Itoa(expected.ServiceDest[0].SrcHttpsPort))
	os.Setenv("DFP_SERVICE_HTTPS_ONLY_2", "false")
	os.Setenv("DFP_SERVICE_PORT_2", expected.ServiceDest[1].Port)
	os.Setenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE_2", expected.ServiceDest[1].ReqPathSearchReplace)
	os.Setenv("DFP_SERVICE_SERVICE_PATH_2", strings.Join(expected.ServiceDest[1].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT_2", strconv.Itoa(expected.ServiceDest[1].SrcPort))
	os.Setenv("DFP_SERVICE_SRC_HTTPS_PORT_2", strconv.Itoa(expected.ServiceDest[1].SrcHttpsPort))
	os.Setenv("DFP_SERVICE_OUTBOUND_HOSTNAME_2", expected.ServiceDest[1].OutboundHostname)
	os.Setenv("DFP_SERVICE_ALLOWED_METHODS_2", strings.Join(expected.ServiceDest[1].AllowedMethods, ","))
	os.Setenv("DFP_SERVICE_DENIED_METHODS_2", strings.Join(expected.ServiceDest[1].DeniedMethods, ","))
	os.Setenv("DFP_SERVICE_REDIRECT_FROM_DOMAIN_2", strings.Join(expected.ServiceDest[1].RedirectFromDomain, ","))
	os.Setenv("DFP_SERVICE_SERVICE_PATH_EXCLUDE_2", strings.Join(expected.ServiceDest[1].ServicePathExclude, ","))
	os.Setenv("DFP_SERVICE_SSL_VERIFY_NONE_2", strconv.FormatBool(expected.ServiceDest[1].VerifyClientSsl))
	os.Setenv("DFP_SERVICE_DENY_HTTP_2", strconv.FormatBool(expected.ServiceDest[1].DenyHttp))
	os.Setenv("DFP_SERVICE_IGNORE_AUTHORIZATION_2", strconv.FormatBool(expected.ServiceDest[1].IgnoreAuthorization))

	defer func() {
		os.Unsetenv("DFP_SERVICE_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_HTTPS_ONLY_1")
		os.Unsetenv("DFP_SERVICE_PORT_1")
		os.Unsetenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE_1")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_1")
		os.Unsetenv("DFP_SERVICE_SRC_PORT_1")
		os.Unsetenv("DFP_SERVICE_SRC_HTTPS_PORT_1")
		os.Unsetenv("DFP_SERVICE_HTTPS_ONLY_2")
		os.Unsetenv("DFP_SERVICE_PORT_2")
		os.Unsetenv("DFP_SERVICE_REQ_PATH_SEARCH_REPLACE_2")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_2")
		os.Unsetenv("DFP_SERVICE_SRC_PORT_2")
		os.Unsetenv("DFP_SERVICE_SRC_HTTPS_PORT_2")
		os.Unsetenv("DFP_SERVICE_OUTBOUND_HOSTNAME_2")
		os.Unsetenv("DFP_SERVICE_ALLOWED_METHODS_2")
		os.Unsetenv("DFP_SERVICE_DENIED_METHODS_2")
		os.Unsetenv("DFP_SERVICE_REDIRECT_FROM_DOMAIN_2")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_EXCLUDE_2")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN_2")
		os.Unsetenv("DFP_SERVICE_VERIFY_CLIENT_SSL_2")
		os.Unsetenv("DFP_SERVICE_DENY_HTTP_2")
		os.Unsetenv("DFP_SERVICE_IGNORE_AUTHORIZATION_2")
	}()
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	expected.ServiceDest[0].ReqMode = "http"
	expected.ServiceDest[1].ReqMode = "http"
	s.Len(*actual, 1)
	s.Contains(*actual, expected)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsEmptyIfServiceNameIsNotSet() {
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 0)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsMultipleServices() {
	service := proxy.Service{
		ServiceName: "my-ServiceName",
		Replicas:    -1,
		ServiceDest: []proxy.ServiceDest{
			{
				Port: "1111",
				ReqPathSearchReplaceFormatted: []string{},
				ServiceDomain:                 []string{},
				ServicePath:                   []string{"my-path-11", "my-path-12"},
				SrcPort:                       1112,
				SrcHttpsPort:                  4443,
				ReqMode:                       "http",
				AllowedMethods:                []string{},
				DeniedMethods:                 []string{},
				RedirectFromDomain:            []string{},
				ServicePathExclude:            []string{},
			},
		},
	}
	os.Setenv("DFP_SERVICE_1_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_1_PORT", service.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_1_SERVICE_PATH", strings.Join(service.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_1_SRC_PORT", strconv.Itoa(service.ServiceDest[0].SrcPort))
	os.Setenv("DFP_SERVICE_1_SRC_HTTPS_PORT", strconv.Itoa(service.ServiceDest[0].SrcHttpsPort))
	defer func() {
		os.Unsetenv("DFP_SERVICE_1_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_1_PORT")
		os.Unsetenv("DFP_SERVICE_1_SERVICE_PATH")
		os.Unsetenv("DFP_SERVICE_1_SRC_PORT")
		os.Unsetenv("DFP_SERVICE_1_SRC_HTTPS_PORT")
	}()

	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

// Mocks

type ReloadMock struct {
	ExecuteMock func(recreate bool) error
}

func MockReload(mock ReloadMock) func() {
	original := actions.NewReload
	actions.NewReload = func() actions.Reloader {
		return &mock
	}
	return func() { actions.NewReload = original }
}

func (m ReloadMock) Execute(recreate bool) error {
	return m.ExecuteMock(recreate)
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

type ReconfigureMock struct {
	ExecuteMock      func(reloadAfter bool) error
	GetDataMock      func() (actions.BaseReconfigure, proxy.Service)
	GetTemplatesMock func() (front, back string, err error)
}

func (m ReconfigureMock) Execute(reloadAfter bool) error {
	return m.ExecuteMock(reloadAfter)
}

func (m ReconfigureMock) GetData() (actions.BaseReconfigure, proxy.Service) {
	return m.GetDataMock()
}

func (m ReconfigureMock) GetTemplates() (front, back string, err error) {
	return m.GetTemplatesMock()
}

type FetchMock struct {
	ReloadServicesFromRegistryMock func(addresses []string, instanceName string) error
	ReloadClusterConfigMock        func(listenerAddr string) error
	ReloadConfigMock               func(baseData actions.BaseReconfigure, listenerAddr string) error
}

func (m *FetchMock) ReloadServicesFromRegistry(addresses []string, instanceName string) error {
	return m.ReloadServicesFromRegistryMock(addresses, instanceName)
}
func (m FetchMock) ReloadClusterConfig(listenerAddr string) error {
	return m.ReloadClusterConfigMock(listenerAddr)
}
func (m FetchMock) ReloadConfig(baseData actions.BaseReconfigure, listenerAddr string) error {
	return m.ReloadConfigMock(baseData, listenerAddr)
}
func MockFetch(mock FetchMock) func() {
	newFetchOrig := actions.NewFetch
	actions.NewFetch = func(baseData actions.BaseReconfigure) actions.Fetchable {
		return &mock
	}
	return func() {
		actions.NewFetch = newFetchOrig
	}
}

type CertMock struct {
	PutMock     func(http.ResponseWriter, *http.Request) (string, error)
	PutCertMock func(certName string, certContent []byte) (string, error)
	GetAllMock  func(w http.ResponseWriter, req *http.Request) (CertResponse, error)
	InitMock    func() error
}

func (m CertMock) Put(w http.ResponseWriter, req *http.Request) (string, error) {
	return m.PutMock(w, req)
}

func (m CertMock) PutCert(certName string, certContent []byte) (string, error) {
	return m.PutCertMock(certName, certContent)
}

func (m CertMock) GetAll(w http.ResponseWriter, req *http.Request) (CertResponse, error) {
	return m.GetAllMock(w, req)
}

func (m CertMock) Init() error {
	return m.InitMock()
}
