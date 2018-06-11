package proxy

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type UtilTestSuite struct {
	suite.Suite
}

func (s *UtilTestSuite) SetupTest() {}

func TestUtilUnitTestSuite(t *testing.T) {
	os.Setenv("SEPARATOR", ",")
	suite.Run(t, new(UtilTestSuite))
}

// cmdRunHa

func (s *UtilTestSuite) Test_HaProxyCmd_DoesNotReturnErrorWhenStdErrIsEmpty() {
	haProxyCmdOrig := haProxyCmd
	defer func() { haProxyCmd = haProxyCmdOrig }()
	haProxyCmd = "ls"
	err := cmdRunHa([]string{"-l"})

	s.NoError(err)
}

func (s *UtilTestSuite) Test_HaProxyCmd_ReturnsError_WhenStdErrIsNotEmpty() {
	haProxyCmdOrig := haProxyCmd
	defer func() { haProxyCmd = haProxyCmdOrig }()
	haProxyCmd = "ls"
	err := cmdRunHa([]string{"-j"}) // `-j` argument is invalid

	s.Error(err)
}

func (s *UtilTestSuite) Test_WaitForPidToUpdate_WaitsForUpdate() {
	readPidFileOrig := readPidFile
	defer func() {
		readPidFile = readPidFileOrig
	}()

	readPidFileCalledCnt := 0
	readPidFile = func(filename string) ([]byte, error) {
		readPidFileCalledCnt++
		if readPidFileCalledCnt == 2 {
			return []byte("102"), nil
		}
		return []byte("101"), nil
	}

	timer := time.NewTimer(2 * time.Second).C
	done := make(chan struct{})

	previousPid := []byte("101")

	go func() {
		waitForPidToUpdate(previousPid, "filename")
		done <- struct{}{}
	}()

L:
	for {
		select {
		case <-timer:
			s.FailNow("Timeout")
		case <-done:
			break L
		}
	}

	s.Equal(2, readPidFileCalledCnt)
}
