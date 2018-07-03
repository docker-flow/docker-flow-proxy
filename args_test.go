// +build !integration

package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ArgsTestSuite struct {
	suite.Suite
	args args
}

func (s *ArgsTestSuite) SetupTest() {
	httpListenAndServe = func(addr string, handler http.Handler) error {
		return nil
	}
}

// NewArgs

func (s ArgsTestSuite) Test_NewArgs_ReturnsNewStruct() {
	a := newArgs()

	s.IsType(args{}, a)
}

// Parse > Server

func (s ArgsTestSuite) Test_Parse_ParsesServerLongArgs() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromArgs", "ip", &serverImpl.IP},
		{"portFromArgs", "port", &serverImpl.Port},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("--%s", d.key), d.expected)
	}
	args{}.parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
	s.Len(serverImpl.ListenerAddresses, 1)
}

func (s ArgsTestSuite) Test_Parse_ParsesServerShortArgs() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromArgs", "i", &serverImpl.IP},
		{"portFromArgs", "p", &serverImpl.Port},
	}

	for _, d := range data {
		os.Args = append(os.Args, fmt.Sprintf("-%s", d.key), d.expected)
	}
	args{}.parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
	s.Len(serverImpl.ListenerAddresses, 1)
}

func (s ArgsTestSuite) Test_Parse_ServerHasDefaultValues() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		value    *string
	}{
		{"0.0.0.0", &serverImpl.IP},
		{"8080", &serverImpl.Port},
	}

	args{}.parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}
	s.Len(serverImpl.ListenerAddresses, 1)
}

func (s ArgsTestSuite) Test_Parse_ServerDefaultsToEnvVars() {
	os.Args = []string{"myProgram", "server"}
	data := []struct {
		expected string
		key      string
		value    *string
	}{
		{"ipFromEnv", "IP", &serverImpl.IP},
		{"portFromEnv", "PORT", &serverImpl.Port},
	}

	for _, d := range data {
		os.Setenv(d.key, d.expected)
	}
	defer func() {
		for _, d := range data {
			os.Unsetenv(d.key)
		}
	}()

	args{}.parse()
	for _, d := range data {
		s.Equal(d.expected, *d.value)
	}

	s.Len(serverImpl.ListenerAddresses, 1)
}

func (s ArgsTestSuite) Test_Parse_ParsesListnerAddressShortArgs() {
	dataTable := []struct {
		value    []string
		expected []string
	}{
		{[]string{"-l", "dfsl1", "-l", "dfsl2"}, []string{"dfsl1", "dfsl2"}},
		{[]string{"-l", "dfsl1"}, []string{"dfsl1"}},
	}

	rootArgs := []string{"myProgram", "server"}
	for _, data := range dataTable {
		os.Args = append(rootArgs, data.value...)
		args{}.parse()
		s.Require().Equal(data.expected, serverImpl.ListenerAddresses)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesListnerAddressLongArgs() {
	dataTable := []struct {
		value    []string
		expected []string
	}{
		{[]string{"--listener-address", "dfsl1", "--listener-address", "dfsl2"}, []string{"dfsl1", "dfsl2"}},
		{[]string{"--listener-address", "dfsl1"}, []string{"dfsl1"}},
	}

	rootArgs := []string{"myProgram", "server"}
	for _, data := range dataTable {
		os.Args = append(rootArgs, data.value...)
		args{}.parse()
		s.Require().Equal(data.expected, serverImpl.ListenerAddresses)
	}
}

func (s ArgsTestSuite) Test_Parse_ParsesListnerAddressEnvVars() {
	os.Args = []string{"myProgram", "server"}
	dataTable := []struct {
		value    string
		expected []string
	}{
		{"dfsl1,dfsl2", []string{"dfsl1", "dfsl2"}},
		{"dfsl1", []string{"dfsl1"}},
	}

	defer func() {
		os.Unsetenv("LISTENER_ADDRESS")
	}()

	for _, data := range dataTable {
		os.Setenv("LISTENER_ADDRESS", data.value)
		args{}.parse()
		s.Require().Equal(data.expected, serverImpl.ListenerAddresses)
	}

}

// Suite

func TestArgsUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, new(ArgsTestSuite))
}
