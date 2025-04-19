package scmrexec

import (
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
)

const (
  ServiceDeleteAccess uint32 = ServiceDelete
  ServiceModifyAccess uint32 = ServiceQueryConfig | ServiceChangeConfig | ServiceStop | ServiceStart | ServiceDelete
  ServiceCreateAccess uint32 = ScManagerCreateService | ServiceStart | ServiceStop | ServiceDelete
  ServiceAllAccess    uint32 = ServiceCreateAccess | ServiceModifyAccess
)

type service struct {
  name           string
  handle         *svcctl.Handle
  originalConfig *svcctl.QueryServiceConfigW
}
