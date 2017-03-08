package logging

import (
"github.com/stretchr/testify/suite"
"testing"
	"fmt"
	"log/syslog"
	"log"
	"time"
)

type LoggingTestSuite struct {
	suite.Suite
}

func TestLoggingUnitTestSuite(t *testing.T) {
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, new(LoggingTestSuite))
}

func (s *LoggingTestSuite) SetupTest() {}

//

func (s LoggingTestSuite) Test_StartLogging_OutputsSyslogToStdOut() {
	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	actual := ""
	logPrintf = func(format string, v ...interface{}) {
		actual = fmt.Sprintf(format, v...)
	}

	go StartLogging()

	for i := 1; i <= 3; i++ {
		sysLog, err := syslog.Dial("udp", "127.0.0.1:1514", syslog.LOG_INFO, "testing")
		if err != nil {
			log.Fatal(err)
		}
		expected := fmt.Sprintf("This is a syslog message %d", i)
		sysLog.Info(expected)
		time.Sleep(10 * time.Millisecond)
		s.Contains(actual, expected)
	}
}