package scmrexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/FalconOpsLLC/goexec/internal/windows"
  "github.com/FalconOpsLLC/goexec/pkg/goexec/dce"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/midl/uuid"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
  "github.com/rs/zerolog"
)

const (
  ModuleName = "SCMR"

  DefaultEndpoint = "ncacn_np:[svcctl]"
  ScmrUuid        = "367ABB81-9844-35F1-AD32-98F038001003"
)

type Scmr struct {
  client *dce.Client
  ctl    svcctl.SvcctlClient
  scm    *svcctl.Handle

  hostname string
}

func (m *Scmr) Init(ctx context.Context, c *dce.Client) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", ModuleName).Logger()

  m.client = c

  if m.client.Dce() == nil {
    return errors.New("DCE connection not initialized")
  }

  m.hostname, err = c.Target.Hostname(ctx)
  if err != nil {
    log.Debug().Err(err).Msg("Failed to determine target hostname")
  }
  if m.hostname == "" {
    m.hostname = util.RandomHostname()
  }

  m.ctl, err = svcctl.NewSvcctlClient(ctx, m.client.Dce(), dcerpc.WithObjectUUID(uuid.MustParse(ScmrUuid)))
  if err != nil {
    log.Error().Err(err).Msg("Failed to initialize SVCCTL client")
    return fmt.Errorf("create SVCCTL client: %w", err)
  }
  log.Info().Msg("Created SVCCTL client")

  resp, err := m.ctl.OpenSCMW(ctx, &svcctl.OpenSCMWRequest{
    MachineName:   m.hostname,
    DatabaseName:  "ServicesActive\x00",
    DesiredAccess: ServiceAllAccess, // TODO: Replace
  })
  if err != nil {
    log.Debug().Err(err).Msg("Failed to open SCM handle")
    return fmt.Errorf("open SCM handle: %w", err)
  }
  log.Info().Msg("Opened SCM handle")

  m.scm = resp.SCM

  return
}

func (m *Scmr) Reconnect(ctx context.Context) (err error) {

  if err = m.client.Reconnect(ctx); err != nil {
    return fmt.Errorf("reconnect: %w", err)
  }
  if err = m.Init(ctx, m.client); err != nil {
    return fmt.Errorf("reconnect SCMR: %w", err)
  }
  return
}

// openService will a handle to the desired service
func (m *Scmr) openService(ctx context.Context, name string) (svc *service, err error) {

  log := zerolog.Ctx(ctx)

  resp, err := m.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
    ServiceManager: m.scm,
    ServiceName:    name,
    DesiredAccess:  ServiceAllAccess, // TODO: dynamic
  })
  if err != nil {
    log.Error().Err(err).Msg("Failed to open service handle")
    return nil, fmt.Errorf("open service: %w", err)
  }

  log.Info().Msg("Opened service handle")

  svc = new(service)
  svc.name = name
  svc.handle = resp.Service

  return
}

func (m *Scmr) startService(ctx context.Context, svc *service) error {

  log := zerolog.Ctx(ctx)

  sr, err := m.ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: svc.handle})

  if err != nil {

    if errors.Is(err, context.DeadlineExceeded) { // Check if execution timed out (execute "cmd.exe /c notepad" for test case)
      log.Warn().Msg("Service execution deadline exceeded")
      svc.handle = nil
      return nil

    } else if sr.Return == windows.ERROR_SERVICE_REQUEST_TIMEOUT {
      log.Info().Msg("Received request timeout. Execution was likely successful")
      return nil
    }

    log.Error().Err(err).Msg("Failed to start service")
    return fmt.Errorf("start service: %w", err)
  }
  log.Info().Msg("Service started successfully")
  return nil
}
