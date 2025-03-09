package tschexec

import (
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
  "time"
)

type Module struct {
  // dce holds the working DCE connection interface
  dce dcerpc.Conn
  // tsch holds the ITaskSchedulerService client
  tsch itaskschedulerservice.TaskSchedulerServiceClient
}

type MethodRegisterConfig struct {
  NoDelete   bool
  CallDelete bool
  //TaskName    string
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
