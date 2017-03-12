package server

type ReloadParams struct {
	Recreate bool  `schema:"recreate"`
	FromListener bool `schema:"fromListener"`
}

type RemoveParams struct {
	AclName     string `schema:"aclName"`
	Distribute  bool   `schema:"distribute"`
	ServiceName string `schema:"serviceName"`
}

type ReconfigureParams struct {
	Distribute  bool   `schema:"distribute"`
	ServiceName string `schema:"serviceName"`
}

