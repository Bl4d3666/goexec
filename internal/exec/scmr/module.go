package scmrexec

import (
  "context"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
)

const (
  MethodCreate = "create"
  MethodChange = "change"

  CleanupMethodDelete = "delete"
  CleanupMethodRevert = "revert"
)

type Module struct {
  hostname  string // The target hostname
  dce       dcerpc.Conn
  reconnect func(context.Context) error

  ctl      svcctl.SvcctlClient
  scm      *svcctl.Handle
  services []remoteService
}

type MethodCreateConfig struct {
  NoDelete    bool
  ServiceName string
  DisplayName string
  ServiceType uint32
  StartType   uint32
}

type MethodChangeConfig struct {
  NoStart     bool
  ServiceName string
}

type CleanupMethodDeleteConfig struct {
  ServiceNames []string
}

type CleanupMethodRevertConfig struct{}
