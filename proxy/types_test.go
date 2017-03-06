package proxy

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type TypesTestSuite struct {
	suite.Suite
}

func (s *TypesTestSuite) SetupTest() {
	logPrintf = func(format string, v ...interface{}) {}
}

// NewRun

func (s TypesTestSuite) Test_ExtractUsersFromString() {

	users := ExtractUsersFromString("sn","u:p", false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "p", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u:p", true, false)
	s.Equal(users, []*User{
		{PassEncrypted: true, Password: "p", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u:p:2", true, false)
	s.Equal(users, []*User{
		{PassEncrypted: true, Password: "p:2", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u", false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u:p,ww", false, true)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "p", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u:p,ww:,:asd", false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "p", Username: "u"},
	})

	users = ExtractUsersFromString("sn","u   ,    uu     ", false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "", Username: "u"},
		{PassEncrypted: false, Password: "", Username: "uu"},
	})

	users = ExtractUsersFromString("sn","", false, false)
	s.Equal(users, []*User{
	})

	users = ExtractUsersFromString("sn",`u   ,
	 uu     `, false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "", Username: "u"},
		{PassEncrypted: false, Password: "", Username: "uu"},
	})
	users = ExtractUsersFromString("sn",`u
uu`, false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "", Username: "u"},
		{PassEncrypted: false, Password: "", Username: "uu"},
	})


	users = ExtractUsersFromString("sn",
`u:p
uu:pp,
uuu:ppp

,

x:X`, false, false)
	s.Equal(users, []*User{
		{PassEncrypted: false, Password: "p", Username: "u"},
		{PassEncrypted: false, Password: "pp", Username: "uu"},
		{PassEncrypted: false, Password: "ppp", Username: "uuu"},
		{PassEncrypted: false, Password: "X", Username: "x"},
	})
}

// Suite

func TestRunUnitTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}
