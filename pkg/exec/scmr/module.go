package scmrexec

import (
	"github.com/FalconOpsLLC/goexec/pkg/client/dcerpc"
	"github.com/bryanmcnulty/adauth"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/rs/zerolog"
)

type Module struct {
	creds    *adauth.Credential
	target   *adauth.Target
	hostname string

	log zerolog.Logger
	dce *dcerpc.DCEClient
	ctl svcctl.SvcctlClient
}

type MethodCreateConfig struct {
	NoDelete    bool
	ServiceName string
	DisplayName string
	ServiceType uint32
	StartType   uint32
}

type MethodModifyConfig struct {
	NoStart     bool
	ServiceName string
}
