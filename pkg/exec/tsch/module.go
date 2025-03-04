package tschexec

import (
	"context"
	"github.com/FalconOpsLLC/goexec/pkg/client/dcerpc"
	"github.com/RedTeamPentesting/adauth"
	"github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
	"github.com/rs/zerolog"
	"time"
)

type Step struct {
	Name   string                                                      // Name of the step
	Status string                                                      // Status indicates whether the task succeeded, failed, etc.
	Call   func(context.Context, *Module, ...any) (interface{}, error) // Call will invoke the procedure
	Match  func(context.Context, *Module, ...any) (bool, error)        // Match will make an assertion to determine whether the step was successful
}

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
