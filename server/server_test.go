package server

import (
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"../proxy"
)

type ServerTestSuite struct {
	BaseUrl        string
	ReconfigureUrl string
	ServiceName    string
	Server         *httptest.Server
	DnsIps         []string
	suite.Suite
}

func (s *ServerTestSuite) SetupTest() {
}

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	s.ServiceName = "my-fancy-service"
	s.BaseUrl = "/v1/docker-flow-proxy/reconfigure"
	s.ReconfigureUrl = fmt.Sprintf(
		"%s?serviceName=%s&serviceColor=pink&servicePath=/path/to/my/service/api&serviceDomain=my-domain.com",
		s.BaseUrl,
		s.ServiceName,
	)

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/reconfigure":
				if strings.EqualFold(r.URL.Query().Get("returnError"), "true") {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
				}
			case "/v1/docker-flow-proxy/remove":
				if strings.EqualFold(r.URL.Query().Get("returnError"), "true") {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { s.Server.Close() }()

	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return s.DnsIps, nil
	}

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	addr := strings.Replace(s.Server.URL, "http://", "", -1)
	s.DnsIps = []string{strings.Split(addr, ":")[0]}

	suite.Run(t, s)
}

// SendDistributeRequests

func (s *ServerTestSuite) Test_SendDistributeRequests_InvokesLookupHost() {
	var actualHost string
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		actualHost = host
		return []string{}, nil
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{}
	srv.SendDistributeRequests(req, "8080", s.ServiceName)

	s.Assert().Equal(fmt.Sprintf("tasks.%s", s.ServiceName), actualHost)
}

func (s *ServerTestSuite) Test_SednDistributeRequests_ReturnsError_WhenLookupHostFails() {
	lookupHostOrig := lookupHost
	defer func() { lookupHost = lookupHostOrig }()
	lookupHost = func(host string) (addrs []string, err error) {
		return []string{}, fmt.Errorf("This is an LookupHost error")
	}
	req, _ := http.NewRequest("GET", s.ReconfigureUrl, nil)

	srv := Serve{}
	status, err := srv.SendDistributeRequests(req, "8080", s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIp() {
	var actualPath string
	var actualQuery url.Values
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualQuery = r.URL.Query()
		actualPath = r.URL.Path
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal(s.BaseUrl, actualPath)
	s.Assert().Equal("false", actualQuery.Get("distribute"))
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIpWithTheCorrectMethod() {
	actualProtocol := "GET"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualProtocol = r.Method
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, nil)

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal("PUT", actualProtocol)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_SendsHttpRequestForEachIpWithTheBody() {
	actualBody := ""
	expectedBody := "THIS IS BODY"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		actualBody = string(body)
	}))
	defer func() { testServer.Close() }()
	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("PUT", addr, strings.NewReader(expectedBody))

	srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assert().Equal(expectedBody, actualBody)
}

func (s *ServerTestSuite) Test_SendDistributeRequests_ReturnsError_WhenRequestFail() {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer func() { testServer.Close() }()

	tsAddr := strings.Replace(testServer.URL, "http://", "", -1)
	dnsIpsOrig := s.DnsIps
	defer func() { s.DnsIps = dnsIpsOrig }()
	s.DnsIps = []string{strings.Split(tsAddr, ":")[0]}
	port := strings.Split(tsAddr, ":")[1]

	srv := Serve{}
	addr := fmt.Sprintf("http://initial-proxy-address:%s%s&distribute=true", port, s.ReconfigureUrl)
	req, _ := http.NewRequest("GET", addr, nil)

	status, err := srv.SendDistributeRequests(req, port, s.ServiceName)

	s.Assertions.Equal(http.StatusBadRequest, status)
	s.Assertions.Error(err)
}

// GetServiceFromUrl

func (s *ServerTestSuite) Test_GetServiceFromUrl_ReturnsProxyService() {
	expected := proxy.Service{
		ServiceName:          "serviceName",
		AclName:              "aclName",
		ServiceColor:         "serviceColor",
		ServiceCert:          "serviceCert",
		OutboundHostname:     "outboundHostname",
		ConsulTemplateFePath: "consulTemplateFePath",
		ConsulTemplateBePath: "consulTemplateBePath",
		PathType:             "pathType",
		ReqPathSearch:        "reqPathSearch",
		ReqPathReplace:       "reqPathReplace",
		TemplateFePath:       "templateFePath",
		TemplateBePath:       "templateBePath",
		TimeoutServer:        "timeoutServer",
		TimeoutTunnel:        "timeoutTunnel",
		ReqMode:              "reqMode",
		HttpsOnly:            true,
		XForwardedProto:	  true,
		RedirectWhenHttpProto: true,
		HttpsPort:			  1234,
		ServiceDomain:        []string{"domain1", "domain2"},
		SkipCheck:				true,
		Distribute:				true,
		SslVerifyNone:			true,
		ServiceDomainMatchAll:  true,
		ServiceDest:          []proxy.ServiceDest{},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&aclName=%s&serviceColor=%s&serviceCert=%s&outboundHostname=%s&consulTemplateFePath=%s&consulTemplateBePath=%s&pathType=%s&reqPathSearch=%s&reqPathReplace=%s&templateFePath=%s&templateBePath=%s&timeoutServer=%s&timeoutTunnel=%s&reqMode=%s&httpsOnly=%t&xForwardedProto=%t&redirectWhenHttpProto=%t&httpsPort=%d&serviceDomain=%s&skipCheck=%t&distribute=%t&sslVerifyNone=%t&serviceDomainMatchAll=%t",
		s.BaseUrl,
		expected.ServiceName,
		expected.AclName,
		expected.ServiceColor,
		expected.ServiceCert,
		expected.OutboundHostname,
		expected.ConsulTemplateFePath,
		expected.ConsulTemplateBePath,
		expected.PathType,
		expected.ReqPathSearch,
		expected.ReqPathReplace,
		expected.TemplateFePath,
		expected.TemplateBePath,
		expected.TimeoutServer,
		expected.TimeoutTunnel,
		expected.ReqMode,
		expected.HttpsOnly,
		expected.XForwardedProto,
		expected.RedirectWhenHttpProto,
		expected.HttpsPort,
		strings.Join(expected.ServiceDomain, ","),
		expected.SkipCheck,
		expected.Distribute,
		expected.SslVerifyNone,
		expected.ServiceDomainMatchAll,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Equal(expected, actual)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_DefaultsReqModeToHttp() {
	req, _ := http.NewRequest("GET", s.BaseUrl, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Equal("http", actual.ReqMode)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_GetsUsersFromProxyExtractUsersFromString() {
	user1 := proxy.User{
		Username: "user1",
		Password: "pass1",
		PassEncrypted: false,
	}
	user2 := proxy.User{
		Username: "user2",
		Password: "pass2",
		PassEncrypted: true,
	}
	extractUsersFromStringOrig := extractUsersFromString
	defer func() { extractUsersFromString = extractUsersFromStringOrig }()
	actualServiceName := ""
	expectedServiceName := "my-service"
	actualUsers := ""
	expectedUsers := "user1,user2"
	actualUsersPassEncrypted := false
	expectedUsersPassEncrypted := true
	actualSkipEmptyPassword := true
	expectedSkipEmptyPassword := false
	extractUsersFromString = func(serviceName, usersString string, usersPassEncrypted, skipEmptyPassword bool) ([]*proxy.User) {
		actualServiceName = serviceName
		actualUsers = usersString
		actualUsersPassEncrypted = usersPassEncrypted
		actualSkipEmptyPassword = skipEmptyPassword
		return []*proxy.User{&user1, &user2}
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&users=%s&usersPassEncrypted=%t",
		s.BaseUrl,
		expectedServiceName,
		expectedUsers,
		expectedUsersPassEncrypted,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := Serve{}

	actual := srv.GetServiceFromUrl([]proxy.ServiceDest{}, req)

	s.Contains(actual.Users, user1)
	s.Contains(actual.Users, user2)
	s.Equal(expectedServiceName, actualServiceName)
	s.Equal(expectedUsers, actualUsers)
	s.Equal(expectedUsersPassEncrypted, actualUsersPassEncrypted)
	s.Equal(expectedSkipEmptyPassword, actualSkipEmptyPassword)
}

// mergeUsers

func (s *ServerTestSuite) Test_UsersMerge_AllCases() {
	usersBasePathOrig := usersBasePath
	defer func() { usersBasePath = usersBasePathOrig }()
	usersBasePath = "../test_configs/%s.txt"
	users := mergeUsers("someService", "user1:pass1,user2:pass2", "", false, "", false)
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})
	users = mergeUsers("someService", "user1:pass1,user2", "", false, "", false)
	//user without password will not be included
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
	})
	users = mergeUsers("someService", "user1:passWoRd,user2", "users", false, "", false)
	//user2 password will come from users file
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "users", true, "", false)
	//user2 password will come from users file, all encrypted
	s.Equal(users, []proxy.User{
		{PassEncrypted: true, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "users", false, "user1:pass1,user2:pass2", false)
	//user2 password will come from users file, but not from global one
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})


	users = mergeUsers("someService", "user1:passWoRd,user2", "", false, "user1:pass1,user2:pass2", false)
	//user2 password will come from global file
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "user1:passWoRd,user2", "", false, "user1:pass1,user2:pass2", true)
	//user2 password will come from global file, globals encrypted only
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})


	users = mergeUsers("someService", "user1:passWoRd,user2", "", true, "user1:pass1,user2:pass2", true)
	//user2 password will come from global file, all encrypted
	s.Equal(users, []proxy.User{
		{PassEncrypted: true, Password: "passWoRd", Username: "user1"},
		{PassEncrypted: true, Password: "pass2", Username: "user2"},
	})


	users = mergeUsers("someService", "user1,user2", "", false, "", false)
	//no users found dummy one generated
	s.Equal(len(users), 1)
	s.Equal(users[0].Username, "dummyUser")


	users = mergeUsers("someService", "", "users", false, "", false)
	//Users from file only
	s.Equal(users, []proxy.User{
		{PassEncrypted: false, Password: "pass1", Username: "user1"},
		{PassEncrypted: false, Password: "pass2", Username: "user2"},
	})

	users = mergeUsers("someService", "", "", false, "user1:pass1,user2:pass2", false)
	//No users when only globals present
	s.Equal(len(users), 0)

}

// Mocks

type ServerMock struct {
	mock.Mock
}

func (m *ServerMock) SendDistributeRequests(req *http.Request, port, serviceName string) (status int, err error) {
	params := m.Called(req, port, serviceName)
	return params.Int(0), params.Error(1)
}

func (m *ServerMock) GetServiceFromUrl(sd []proxy.ServiceDest, req *http.Request) proxy.Service {
	params := m.Called(sd, req)
	return params.Get(0).(proxy.Service)
}

func getServerMock(skipMethod string) *ServerMock {
	mockObj := new(ServerMock)
	if skipMethod != "SendDistributeRequests" {
		mockObj.On("SendDistributeRequests", mock.Anything, mock.Anything, mock.Anything).Return(200, nil)
	}
	return mockObj
}
