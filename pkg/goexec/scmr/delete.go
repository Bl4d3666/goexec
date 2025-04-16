package scmrexec

import (
  "context"
  "fmt"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
  "github.com/rs/zerolog"
)

const (
  MethodDelete = "Delete"
)

type ScmrDelete struct {
  Scmr
  goexec.Cleaner

  ServiceName string
}

func (m *ScmrDelete) Clean(ctx context.Context) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", ModuleName).
    Str("method", MethodDelete).
    Str("service", m.ServiceName).
    Logger()

  svc, err := m.openService(ctx, m.ServiceName)
  if err != nil {
    return err
  }
  deleteResponse, err := m.ctl.DeleteService(ctx, &svcctl.DeleteServiceRequest{
    Service: svc.handle,
  })
  if err != nil {
    return fmt.Errorf("delete service: %w", err)
  }
  if deleteResponse.Return != 0 {
    return fmt.Errorf("delete service returned non-zero exit code: %02x", deleteResponse.Return)
  }

  log.Info().Msg("Deleted service")
  return
}
