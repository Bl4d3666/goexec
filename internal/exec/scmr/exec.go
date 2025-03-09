package scmrexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/FalconOpsLLC/goexec/internal/windows"
  "github.com/RedTeamPentesting/adauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/midl/uuid"
  "github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
  "github.com/rs/zerolog"
)

const (
  DefaultEndpoint = "ncacn_np:[srvsvc]"
)

var (
  ScmrRpcUuid = uuid.MustParse("367ABB81-9844-35F1-AD32-98F038001003")
)

func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.ConnectionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("func", "Connect").Logger()

  if ccfg.ConnectionMethod == exec.ConnectionMethodDCE {
    if cfg, ok := ccfg.ConnectionMethodConfig.(dce.ConnectionMethodDCEConfig); !ok {
      return fmt.Errorf("invalid configuration for DCE connection method")
    } else {

      // Fetch target hostname - for opening SCM handle
      if mod.hostname, err = target.Hostname(ctx); err != nil {
        log.Debug().Err(err).Msg("Failed to get target hostname")
        mod.hostname = util.RandomHostname()
        err = nil
      }
      connect := func(ctx context.Context) error {
        // Create DCE connection
        if mod.dce, err = cfg.GetDce(ctx, creds, target, dcerpc.WithObjectUUID(ScmrRpcUuid)); err != nil {
          log.Error().Err(err).Msg("Failed to initialize DCE dialer")
          return fmt.Errorf("create DCE dialer: %w", err)
        }
        log.Info().Msg("DCE dialer initialized")

        // Create SVCCTL client
        mod.ctl, err = svcctl.NewSvcctlClient(ctx, mod.dce)
        if err != nil {
          log.Error().Err(err).Msg("Failed to initialize SCMR client")
          return fmt.Errorf("init SCMR client: %w", err)
        }
        log.Info().Msg("DCE connection successful")
        return nil
      }
      mod.reconnect = func(c context.Context) error {
        mod.dce = nil
        mod.ctl = nil
        return connect(c)
      }
      return connect(ctx)
    }
  } else {
    return errors.New("unsupported connection method")
  }
}

func (mod *Module) Cleanup(ctx context.Context, ccfg *exec.CleanupConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("method", ccfg.CleanupMethod).
    Str("func", "Cleanup").Logger()

  if len(mod.services) == 0 {
    return nil
  }

  if mod.dce == nil || mod.ctl == nil {
    // Try to reconnect
    if err := mod.reconnect(ctx); err != nil {
      log.Error().Err(err).Msg("Reconnect failed")
      return err
    }
    log.Info().Msg("Reconnect successful")
  }
  if mod.scm == nil {
    // Open a handle to SCM (again)
    if resp, err := mod.ctl.OpenSCMW(ctx, &svcctl.OpenSCMWRequest{
      MachineName:   util.CheckNullString(mod.hostname),
      DatabaseName:  "ServicesActive\x00",
      DesiredAccess: ServiceAllAccess, // TODO: Replace
    }); err != nil {
      log.Error().Err(err).Msg("Failed to reopen an SCM handle")
      return err
    } else {
      mod.scm = resp.SCM
      log.Info().Msg("Reopened SCM handle")
    }
  }

  for _, rsvc := range mod.services {
    log = log.With().Str("service", rsvc.name).Logger()

    if rsvc.handle == nil {
      // Open a handle to the service in question
      if or, err := mod.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
        ServiceManager: mod.scm,
        ServiceName:    rsvc.name,
        DesiredAccess:  windows.SERVICE_DELETE | windows.SERVICE_CHANGE_CONFIG,
      }); err != nil {
        log.Error().Err(err).Msg("Failed to reopen a service handle")
        continue
      } else {
        rsvc.handle = or.Service
      }
      log.Info().Msg("Service handle reopened")
    }
    if ccfg.CleanupMethod == CleanupMethodDelete {
      // Delete the service
      if _, err = mod.ctl.DeleteService(ctx, &svcctl.DeleteServiceRequest{Service: rsvc.handle}); err != nil {
        log.Error().Err(err).Msg("Failed to delete service")
        continue
      }
      log.Info().Msg("Service deleted successfully")

    } else if ccfg.CleanupMethod == CleanupMethodRevert {
      // Revert the service configuration & state
      log.Info().Msg("Attempting to revert service configuration")
      if _, err = mod.ctl.ChangeServiceConfigW(ctx, &svcctl.ChangeServiceConfigWRequest{
        Service: rsvc.handle,
        //Dependencies:     []byte(rsvc.originalConfig.Dependencies), // TODO: ensure this works
        ServiceType:      rsvc.originalConfig.ServiceType,
        StartType:        rsvc.originalConfig.StartType,
        ErrorControl:     rsvc.originalConfig.ErrorControl,
        BinaryPathName:   rsvc.originalConfig.BinaryPathName,
        LoadOrderGroup:   rsvc.originalConfig.LoadOrderGroup,
        ServiceStartName: rsvc.originalConfig.ServiceStartName,
        DisplayName:      rsvc.originalConfig.DisplayName,
        TagID:            rsvc.originalConfig.TagID,
      }); err != nil {
        log.Error().Err(err).Msg("Failed to revert service configuration")
        continue
      }
      log.Info().Msg("Service configuration reverted")
    }
    if _, err = mod.ctl.CloseService(ctx, &svcctl.CloseServiceRequest{ServiceObject: rsvc.handle}); err != nil {
      log.Warn().Err(err).Msg("Failed to close service handle")
      return nil
    }
    log.Info().Msg("Closed service handle")
  }
  return
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {

  //vctx := context.WithoutCancel(ctx)
  log := zerolog.Ctx(ctx).With().
    Str("method", ecfg.ExecutionMethod).
    Str("func", "Exec").Logger()

  if ecfg.ExecutionMethod == MethodCreate {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodCreateConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      svc := remoteService{
        name: cfg.ServiceName,
      }
      defer func() { // TODO: relocate this?
        mod.services = append(mod.services, svc)
      }()
      // Open a handle to SCM
      if resp, err := mod.ctl.OpenSCMW(ctx, &svcctl.OpenSCMWRequest{
        MachineName:   util.CheckNullString(mod.hostname),
        DatabaseName:  "ServicesActive\x00",
        DesiredAccess: ServiceAllAccess, // TODO: Replace
      }); err != nil {
        log.Debug().Err(err).Msg("Failed to open SCM handle")
        return fmt.Errorf("open SCM handle: %w", err)
      } else {
        mod.scm = resp.SCM
        log.Info().Msg("Opened SCM handle")
      }
      // Create service
      serviceName := util.RandomStringIfBlank(svc.name)
      resp, err := mod.ctl.CreateServiceW(ctx, &svcctl.CreateServiceWRequest{
        ServiceManager: mod.scm,
        ServiceName:    serviceName,
        DisplayName:    util.RandomStringIfBlank(cfg.DisplayName),
        BinaryPathName: util.CheckNullString(ecfg.GetRawCommand()),
        ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
        StartType:      windows.SERVICE_DEMAND_START,
        DesiredAccess:  ServiceAllAccess, // TODO: Replace
      })
      if err != nil || resp == nil || resp.Return != 0 {
        log.Error().Err(err).Msg("Failed to create service")
        return fmt.Errorf("create service: %w", err)
      }
      svc.handle = resp.Service

      log = log.With().
        Str("service", serviceName).Logger()
      log.Info().Msg("Service created")

      // Start the service
      sr, err := mod.ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: svc.handle})
      if err != nil {

        if errors.Is(err, context.DeadlineExceeded) { // Check if execution timed out (execute "cmd.exe /c notepad" for test case)
          log.Warn().Err(err).Msg("Service execution deadline exceeded")
          // Connection closes, so we nullify the client variables and handles
          mod.dce = nil
          mod.ctl = nil
          mod.scm = nil
          svc.handle = nil

        } else if sr != nil && sr.Return == windows.ERROR_SERVICE_REQUEST_TIMEOUT { // Check for request timeout
          log.Info().Msg("Received request timeout. Execution was likely successful")

        } else {
          log.Error().Err(err).Msg("Failed to start service")
          return fmt.Errorf("start service: %w", err)
        }
        // Inform the caller that execution was likely successful despite error
        err = nil
      } else {
        log.Info().Msg("Started service")
      }
    }
  } else if ecfg.ExecutionMethod == MethodChange {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodChangeConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      svc := remoteService{
        name: cfg.ServiceName,
      }
      defer func() { // TODO: relocate this?
        mod.services = append(mod.services, svc)
      }()

      // Open a handle to SCM
      if resp, err := mod.ctl.OpenSCMW(ctx, &svcctl.OpenSCMWRequest{
        MachineName:   util.CheckNullString(mod.hostname),
        DatabaseName:  "ServicesActive\x00",
        DesiredAccess: ServiceAllAccess, // TODO: Replace
      }); err != nil {
        log.Debug().Err(err).Msg("Failed to open SCM handle")
        return fmt.Errorf("open SCM handle: %w", err)
      } else {
        mod.scm = resp.SCM
        log.Info().Msg("Opened SCM handle")
      }

      // Open a handle to the desired service
      if resp, err := mod.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
        ServiceManager: mod.scm,
        ServiceName:    svc.name,
        DesiredAccess:  ServiceAllAccess, // TODO: Replace
      }); err != nil {
        log.Error().Err(err).Msg("Failed to open service handle")
        return fmt.Errorf("open service: %w", err)
      } else {
        svc.handle = resp.Service
      }

      // Note original service status
      if resp, err := mod.ctl.QueryServiceStatus(ctx, &svcctl.QueryServiceStatusRequest{
        Service: svc.handle,
      }); err != nil {
        log.Warn().Err(err).Msg("Failed to get service status")
      } else {
        svc.originalState = resp.ServiceStatus
      }

      // Note original service configuration
      if resp, err := mod.ctl.QueryServiceConfigW(ctx, &svcctl.QueryServiceConfigWRequest{
        Service:      svc.handle,
        BufferLength: 8 * 1024,
      }); err != nil {
        log.Error().Err(err).Msg("Failed to fetch service configuration")
        return fmt.Errorf("get service config: %w", err)
      } else {
        log.Info().Str("binaryPath", resp.ServiceConfig.BinaryPathName).Msg("Fetched original service configuration")
        svc.originalConfig = resp.ServiceConfig
      }

      // Stop service if its running
      if svc.originalState == nil || svc.originalState.CurrentState != windows.SERVICE_STOPPED {
        if resp, err := mod.ctl.ControlService(ctx, &svcctl.ControlServiceRequest{
          Service: svc.handle,
          Control: windows.SERVICE_STOPPED,
        }); err != nil {
          if resp != nil && resp.Return == windows.ERROR_SERVICE_NOT_ACTIVE {
            log.Info().Msg("Service is already stopped")
          } else {
            log.Error().Err(err).Msg("Failed to stop service")
          }
        } else {
          log.Info().Msg("Service stopped")
        }
      }
      // Change service configuration
      if _, err = mod.ctl.ChangeServiceConfigW(ctx, &svcctl.ChangeServiceConfigWRequest{
        Service:        svc.handle,
        BinaryPathName: ecfg.GetRawCommand(),
        //Dependencies:     []byte(svc.originalConfig.Dependencies), // TODO: ensure this works
        ServiceType:      svc.originalConfig.ServiceType,
        StartType:        windows.SERVICE_DEMAND_START,
        ErrorControl:     svc.originalConfig.ErrorControl,
        LoadOrderGroup:   svc.originalConfig.LoadOrderGroup,
        ServiceStartName: svc.originalConfig.ServiceStartName,
        DisplayName:      svc.originalConfig.DisplayName,
        TagID:            svc.originalConfig.TagID,
      }); err != nil {
        log.Error().Err(err).Msg("Failed to change service configuration")
        return fmt.Errorf("change service configuration: %w", err)
      }
      log.Info().Msg("Successfully altered service configuration")

      if !cfg.NoStart {
        if resp, err := mod.ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: svc.handle}); err != nil {

          if errors.Is(err, context.DeadlineExceeded) { // Check if execution timed out (execute "cmd.exe /c notepad" for test case)
            log.Warn().Err(err).Msg("Service execution deadline exceeded")
            // Connection closes, so we nullify the client variables and handles
            mod.dce = nil
            mod.ctl = nil
            mod.scm = nil
            svc.handle = nil

          } else if resp != nil && resp.Return == windows.ERROR_SERVICE_REQUEST_TIMEOUT { // Check for request timeout
            log.Info().Err(err).Msg("Received request timeout. Execution was likely successful")
          } else {
            log.Error().Err(err).Msg("Failed to start service")
            return fmt.Errorf("start service: %w", err)
          }
        }
      }
    }
  }
  return
}
