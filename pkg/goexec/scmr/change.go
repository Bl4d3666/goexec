package scmrexec

import (
  "context"
  "fmt"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
  "github.com/rs/zerolog"
)

const (
  MethodChange = "Change"
)

type ScmrChange struct {
  Scmr
  goexec.Cleaner
  goexec.Executor

  NoStart     bool
  ServiceName string
}

func (m *ScmrChange) Execute(ctx context.Context, in *goexec.ExecutionInput) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", ModuleName).
    Str("method", MethodChange).
    Str("service", m.ServiceName).
    Logger()

  svc := &service{name: m.ServiceName}

  openResponse, err := m.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
    ServiceManager: m.scm,
    ServiceName:    svc.name,
    DesiredAccess:  ServiceAllAccess,
  })

  if err != nil {
    log.Error().Err(err).Msg("Failed to open service handle")
    return fmt.Errorf("open service request: %w", err)
  }
  if openResponse.Return != 0 {
    log.Error().Err(err).Msg("Failed to open service handle")
    return fmt.Errorf("create service: %w", err)
  }

  svc.handle = openResponse.Service
  log.Info().Msg("Opened service handle")

  defer m.AddCleaner(func(ctxInner context.Context) error {

    r, errInner := m.ctl.CloseService(ctxInner, &svcctl.CloseServiceRequest{
      ServiceObject: svc.handle,
    })
    if errInner != nil {
      return fmt.Errorf("close service: %w", errInner)
    }
    if r.Return != 0 {
      return fmt.Errorf("close service returned non-zero exit code: %02x", r.Return)
    }
    log.Info().Msg("Closed service handle")

    return nil
  })

  // Note original service configuration
  queryResponse, err := m.ctl.QueryServiceConfigW(ctx, &svcctl.QueryServiceConfigWRequest{
    Service:      svc.handle,
    BufferLength: 8 * 1024,
  })

  if err != nil {
    log.Error().Err(err).Msg("Failed to fetch service configuration")
    return fmt.Errorf("get service config: %w", err)
  }
  if queryResponse.Return != 0 {
    log.Error().Err(err).Msg("Failed to query service configuration")
    return fmt.Errorf("query service config: %w", err)
  }

  log.Info().Str("binaryPath", queryResponse.ServiceConfig.BinaryPathName).Msg("Fetched original service configuration")
  svc.originalConfig = queryResponse.ServiceConfig

  stopResponse, err := m.ctl.ControlService(ctx, &svcctl.ControlServiceRequest{
    Service: svc.handle,
    Control: ServiceControlStop,
  })

  if err != nil {
    if stopResponse == nil || stopResponse.Return != ErrorServiceNotActive {

      log.Error().Err(err).Msg("Failed to stop existing service")
      return fmt.Errorf("stop service: %w", err)
    }

    log.Debug().Msg("Service is not running")

    // TODO: restore state
    /*
       defer m.AddCleaner(func(ctxInner context.Context) error {
         // ...
         return nil
       })
    */

  } else {
    log.Info().Msg("Stopped existing service")
  }

  changeResponse, err := m.ctl.ChangeServiceConfigW(ctx, &svcctl.ChangeServiceConfigWRequest{
    Service:          svc.handle,
    BinaryPathName:   in.String(),
    DisplayName:      svc.originalConfig.DisplayName,
    ServiceType:      svc.originalConfig.ServiceType,
    StartType:        ServiceDemandStart,
    ErrorControl:     svc.originalConfig.ErrorControl,
    LoadOrderGroup:   svc.originalConfig.LoadOrderGroup,
    ServiceStartName: svc.originalConfig.ServiceStartName,
    TagID:            svc.originalConfig.TagID,
    //Dependencies:     []byte(svc.originalConfig.Dependencies), // TODO
  })

  if err != nil {
    log.Error().Err(err).Msg("Failed to request service configuration change")
    return fmt.Errorf("change service config request: %w", err)
  }
  if changeResponse.Return != 0 {
    log.Error().Err(err).Msg("Failed to change service configuration")
    return fmt.Errorf("change service config: %w", err)
  }

  if !m.NoStart {

    err = m.startService(ctx, svc)
    if err != nil {
      log.Error().Err(err).Msg("Failed to start service")
    }
  }

  return
}
