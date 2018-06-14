package proxy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TemplateTestSuite struct {
	suite.Suite
}

func TestTemplateUnitTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateTestSuite))
}

// FormatServiceForTemplates

func (s *TemplateTestSuite) Test_FormatServiceForTemplates_DiscoveryTypeDNS_GetsReplicasCnt() {
	lookupHostOrig := LookupHost
	defer func() {
		LookupHost = lookupHostOrig
	}()

	actualHost := ""
	LookupHost = func(host string) ([]string, error) {
		actualHost = host
		return []string{"192.168.1.1", "192.168.1.2"}, nil
	}

	service := Service{
		ServiceName:   "my-service-1",
		PathType:      "path_beg",
		DiscoveryType: "DNS",
		Replicas:      0}

	FormatServiceForTemplates(&service)

	s.Equal(2, service.Replicas)
	s.Equal("tasks.my-service-1", actualHost)
}

func (s *TemplateTestSuite) Test_FormatData_UsesServiceNameForAclName() {
	service := Service{ServiceName: "my-service-1"}

	FormatServiceForTemplates(&service)
	s.Equal("my-service-1", service.AclName)

}

func (s *TemplateTestSuite) Test_FormatData_NoPathType_DefaultsToPath_Beg() {
	service := Service{ServiceName: "my-service-1"}

	FormatServiceForTemplates(&service)
	s.Equal("path_beg", service.PathType)

}

func (s *TemplateTestSuite) Test_FormatData_SrcPort_DefinesSrcPortAclNameAndSrcPortAcl() {

	service := Service{
		ServiceName: "my-service-1",
		ServiceDest: []ServiceDest{
			{SrcPort: 4480, Port: "1111",
				ServicePath: []string{"/path-1"}}}}

	FormatServiceForTemplates(&service)

	s.Require().Len(service.ServiceDest, 1)
	sd := service.ServiceDest[0]

	s.Equal(" srcPort_my-service-14480_0", sd.SrcPortAclName)
	s.Equal("\n    acl srcPort_my-service-14480_0 dst_port 4480", sd.SrcPortAcl)

}
