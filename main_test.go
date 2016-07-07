// +build !integration

package main

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

// Setup

type MainTestSuite struct {
	suite.Suite
}

func (s *MainTestSuite) SetupTest() {}

// main

func (s MainTestSuite) Test_Main_InvokesArgsParse() {
	actual := false
	NewArgs = func() Args {
		actual = true
		return Args{}
	}

	main()

	s.True(actual)
}

// Suite

func TestMainUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, new(MainTestSuite))
}
