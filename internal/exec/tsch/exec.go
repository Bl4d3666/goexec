package tschexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/RedTeamPentesting/adauth"
  "github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
  "github.com/rs/zerolog"
  "time"
)

const (
  TschDefaultEndpoint = "ncacn_np:[atsvc]"
  TschDefaultObject   = "86D35949-83C9-4044-B424-DB363231FD0C"
)

// Connect to the target & initialize DCE & TSCH clients
func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.ConnectionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("func", "Connect").Logger()

  if ccfg.ConnectionMethod == exec.ConnectionMethodDCE {
    if cfg, ok := ccfg.ConnectionMethodConfig.(dce.ConnectionMethodDCEConfig); !ok {
      return fmt.Errorf("invalid configuration for DCE connection method")
    } else {
      // Create DCERPC dialer
      mod.dce, err = cfg.GetDce(ctx, creds, target, TschDefaultEndpoint, TschDefaultObject)
      if err != nil {
        log.Error().Err(err).Msg("Failed to create DCERPC dialer")
        return fmt.Errorf("create DCERPC dialer: %w", err)
      }
      // Create ITaskSchedulerService client
      mod.tsch, err = itaskschedulerservice.NewTaskSchedulerServiceClient(ctx, mod.dce)
      if err != nil {
        log.Error().Err(err).Msg("Failed to initialize TSCH client")
        return fmt.Errorf("init TSCH client: %w", err)
      }
      log.Info().Msg("DCE connection successful")
    }
  } else {
    return errors.New("unsupported connection method")
  }
  return
}

func (mod *Module) Cleanup(ctx context.Context, ccfg *exec.CleanupConfig) (err error) {
  log := zerolog.Ctx(ctx).With().
    Str("method", ccfg.CleanupMethod).
    Str("func", "Cleanup").Logger()

  if ccfg.CleanupMethod == MethodDelete {
    if cfg, ok := ccfg.CleanupMethodConfig.(MethodDeleteConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      log = log.With().Str("task", cfg.TaskPath).Logger()
      log.Info().Msg("Manually deleting task")

      if err = mod.deleteTask(ctx, cfg.TaskPath); err == nil {
        log.Info().Msg("Task deleted successfully")
      }
    }
  } else if ccfg.CleanupMethod == "" {
    return nil
  } else {
    return fmt.Errorf("unsupported cleanup method")
  }
  return
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("method", ecfg.ExecutionMethod).
    Str("func", "Exec").Logger()

  if ecfg.ExecutionMethod == MethodRegister {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodRegisterConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      startTime := time.Now().UTC().Add(cfg.StartDelay)
      stopTime := startTime.Add(cfg.StopDelay)

      tr := taskTimeTrigger{
        StartBoundary: startTime.Format(TaskXMLDurationFormat),
        Enabled:       true,
      }
      tk := newTask(nil, nil, triggers{TimeTriggers: []taskTimeTrigger{tr}}, ecfg.ExecutableName, ecfg.ExecutableArgs)

      if !cfg.NoDelete && !cfg.CallDelete {
        if cfg.StopDelay == 0 {
          cfg.StopDelay = time.Second
        }
        tk.Settings.DeleteExpiredTaskAfter = xmlDuration(cfg.DeleteDelay)
        tk.Triggers.TimeTriggers[0].EndBoundary = stopTime.Format(TaskXMLDurationFormat)
      }
      taskPath := cfg.TaskPath
      if taskPath == "" {
        log.Debug().Msg("Task path not defined. Using random path")
        taskPath = `\` + util.RandomString()
      }
      // The taskPath is changed here to the *actual path returned by SchRpcRegisterTask
      taskPath, err = mod.registerTask(ctx, *tk, taskPath)
      if err != nil {
        return fmt.Errorf("call registerTask: %w", err)
      }

      if !cfg.NoDelete {
        if cfg.CallDelete {
          defer mod.deleteTask(ctx, taskPath)

          log.Info().Dur("ms", cfg.StartDelay).Msg("Waiting for task to run")
          select {
          case <-ctx.Done():
            log.Warn().Msg("Cancelling execution")
            return err
          case <-time.After(cfg.StartDelay + time.Second): // + one second for good measure
            for {
              if stat, err := mod.tsch.GetLastRunInfo(ctx, &itaskschedulerservice.GetLastRunInfoRequest{Path: taskPath}); err != nil {
                log.Warn().Err(err).Msg("Failed to get last run info. Assuming task was executed")
                break
              } else if stat.LastRuntime.AsTime().IsZero() {
                log.Warn().Msg("Task was not yet run. Waiting 10 additional seconds")
                time.Sleep(10 * time.Second)
              } else {
                break
              }
            }
            break
          }
        } else {
          log.Info().Time("when", stopTime).Msg("Task is scheduled to delete")
        }
      }
    }
  } else if ecfg.ExecutionMethod == MethodDemand {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodDemandConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      taskPath := cfg.TaskPath

      if taskPath == "" {
        log.Debug().Msg("Task path not defined. Using random path")
        taskPath = `\` + util.RandomString()
      }

      st := newSettings(true, true, false)
      tk := newTask(st, nil, triggers{}, ecfg.ExecutableName, ecfg.ExecutableArgs)

      // The taskPath is changed here to the *actual path returned by SchRpcRegisterTask
      taskPath, err = mod.registerTask(ctx, *tk, taskPath)
      if err != nil {
        return fmt.Errorf("call registerTask: %w", err)
      }

      if !cfg.NoDelete {
        defer mod.deleteTask(ctx, taskPath)
      }

      var flags uint32

      if cfg.SessionId != 0 {
        flags |= 4
      }
      _, err := mod.tsch.Run(ctx, &itaskschedulerservice.RunRequest{
        Path:      taskPath,
        Flags:     flags,
        SessionID: cfg.SessionId,
      })
      if err != nil {
        log.Error().Str("task", taskPath).Err(err).Msg("Failed to run task")
        return fmt.Errorf("force run task: %w", err)
      }
      log.Info().Str("task", taskPath).Msg("Started task")
    }

  } else {
    return fmt.Errorf("method '%s' not implemented", ecfg.ExecutionMethod)
  }

  return nil
}
