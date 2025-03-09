package scmrexec

import (
  "context"
  "github.com/FalconOpsLLC/goexec/internal/windows"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
)

const (
  ServiceDeleteAccess uint32 = windows.SERVICE_DELETE
  ServiceModifyAccess uint32 = windows.SERVICE_QUERY_CONFIG | windows.SERVICE_CHANGE_CONFIG | windows.SERVICE_STOP | windows.SERVICE_START | windows.SERVICE_DELETE
  ServiceCreateAccess uint32 = windows.SC_MANAGER_CREATE_SERVICE | windows.SERVICE_START | windows.SERVICE_STOP | windows.SERVICE_DELETE
  ServiceAllAccess    uint32 = ServiceCreateAccess | ServiceModifyAccess
)

type remoteService struct {
  name           string
  handle         *svcctl.Handle
  originalConfig *svcctl.QueryServiceConfigW
  originalState  *svcctl.ServiceStatus
}

func (mod *Module) parseServiceDependencies(ctx context.Context, ) (err error) {
  return nil
}
