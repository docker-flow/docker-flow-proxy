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

func TestMainSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
