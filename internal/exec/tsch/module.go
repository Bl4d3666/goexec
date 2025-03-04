package tschexec

import (
	"github.com/FalconOpsLLC/goexec/internal/client/dcerpc"
	"github.com/RedTeamPentesting/adauth"
	"github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
	"github.com/rs/zerolog"
	"time"
)

type Module struct {
	creds  *adauth.Credential
	target *adauth.Target

	log  zerolog.Logger
	dce  *dcerpc.DCEClient
	tsch itaskschedulerservice.TaskSchedulerServiceClient
}

type MethodRegisterConfig struct {
	NoDelete    bool
	CallDelete  bool
	TaskName    string
	TaskPath    string
	StartDelay  time.Duration
	StopDelay   time.Duration
	DeleteDelay time.Duration
}

type MethodDemandConfig struct {
	NoDelete    bool
	CallDelete  bool
	TaskName    string
	TaskPath    string
	StopDelay   time.Duration
	DeleteDelay time.Duration
}

type MethodDeleteConfig struct {
	TaskPath string
}

const (
	MethodRegister string = "register"
	MethodDemand   string = "demand"
	MethodDelete   string = "delete"
	MethodChange   string = "update"
)
