package server

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type CertTestSuite struct {
	suite.Suite
}

func (s *CertTestSuite) SetupTest() {
}

func TestCertUnitTestSuite(t *testing.T) {
	s := new(CertTestSuite)
	suite.Run(t, s)
}

// Put

func (s *CertTestSuite) Test_Put_SavesBodyAsFile() {
	c := NewCert("../certs")
	certName := "test.pem"
	expected := "THIS IS A CERTIFICATE"
	path := fmt.Sprintf("%s/%s", c.CertsDir, certName)
	os.Remove(path)
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=%s", certName),
		strings.NewReader(expected),
	)

	c.Put(w, req)
	actual, err := ioutil.ReadFile(path)

	s.NoError(err)
	s.Equal(expected, string(actual))
}

func (s *CertTestSuite) Test_Put_SetsContentTypeToJson() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem",
		strings.NewReader(""),
	)

	c.Put(w, req)

	s.Equal("application/json", actual)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus200() {
	expected, _ := json.Marshal(CertResponse{
		Status:  "OK",
		Message: "",
	})
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		"http://acme.com/v1/docker-flow-proxy/cert?certName=my-cert.pem",
		strings.NewReader(""),
	)

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 200)
	w.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenDirectoryDoesNotExist() {
	c := NewCert("THIS_PATH_DOES_NOT_EXIST")
	w := getResponseWriterMock()
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", strings.NewReader(""))

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus400_WhenDirectoryDoesNotExist() {
	c := NewCert("THIS_PATH_DOES_NOT_EXIST")
	w := getResponseWriterMock()
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", strings.NewReader(""))

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenCannotReadBody() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	r := ReaderMock{
		ReadMock: func([]byte) (int, error) { return 0, fmt.Errorf("This is an error") },
	}
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", r)

	_, err := c.Put(w, req)

	s.Error(err)
}

func (s *CertTestSuite) Test_Put_WritesHeaderStatus40_WhenCannotReadBody() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	r := ReaderMock{
		ReadMock: func([]byte) (int, error) { return 0, fmt.Errorf("This is an error") },
	}
	req, _ := http.NewRequest("PUT", "http://acme.com/v1/docker-flow-proxy/cert?certName=test.pem", r)

	c.Put(w, req)

	w.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *CertTestSuite) Test_Put_ReturnsCertPath() {
	c := NewCert("../certs")
	certName := "test.pem"
	expected, _ := filepath.Abs(fmt.Sprintf("%s/%s", c.CertsDir, certName))
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert?certName=%s", certName),
		strings.NewReader(""),
	)

	actual, _ := c.Put(w, req)

	s.Equal(expected, actual)
}

func (s *CertTestSuite) Test_Put_ReturnsError_WhenCertNameDoesNotExist() {
	c := NewCert("../certs")
	w := getResponseWriterMock()
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://acme.com/v1/docker-flow-proxy/cert"),
		strings.NewReader(""),
	)

	_, err := c.Put(w, req)

	s.Error(err)
}

// Mock

// ReaderMock

type ReaderMock struct {
	ReadMock func([]byte) (int, error)
}

func (m ReaderMock) Read(p []byte) (int, error) {
	return m.ReadMock(p)
}

// ResponseWriterMock

type ResponseWriterMock struct {
	mock.Mock
}

func (m *ResponseWriterMock) Header() http.Header {
	m.Called()
	return make(map[string][]string)
}

func (m *ResponseWriterMock) Write(data []byte) (int, error) {
	params := m.Called(data)
	return params.Int(0), params.Error(1)
}

func (m *ResponseWriterMock) WriteHeader(header int) {
	m.Called(header)
}

func getResponseWriterMock() *ResponseWriterMock {
	mockObj := new(ResponseWriterMock)
	mockObj.On("Header").Return(nil)
	mockObj.On("Write", mock.Anything).Return(0, nil)
	mockObj.On("WriteHeader", mock.Anything)
	return mockObj
}
