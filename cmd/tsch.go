package cmd

import (
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/FalconOpsLLC/goexec/internal/exec/tsch"
  "github.com/spf13/cobra"
  "time"
)

func tschCmdInit() {
  registerRpcFlags(tschCmd)

  tschDeleteCmdInit()
  tschCmd.AddCommand(tschDeleteCmd)
  tschRegisterCmdInit()
  tschCmd.AddCommand(tschRegisterCmd)
  tschDemandCmdInit()
  tschCmd.AddCommand(tschDemandCmd)
}

func tschDeleteCmdInit() {
  tschDeleteCmd.Flags().StringVarP(&tschTaskPath, "path", "t", "", "Scheduled task path")
  if err := tschDeleteCmd.MarkFlagRequired("path"); err != nil {
    panic(err)
  }
}

func tschDemandCmdInit() {
  tschDemandCmd.Flags().StringVarP(&executable, "executable", "e", "", "Remote Windows executable to invoke")
  tschDemandCmd.Flags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
  tschDemandCmd.Flags().StringVarP(&tschName, "name", "n", "", "Target task name")
  tschDemandCmd.Flags().BoolVar(&tschNoDelete, "no-delete", false, "Don't delete task after execution")
  if err := tschDemandCmd.MarkFlagRequired("executable"); err != nil {
    panic(err)
  }
}

func tschRegisterCmdInit() {
  tschRegisterCmd.Flags().StringVarP(&executable, "executable", "e", "", "Remote Windows executable to invoke")
  tschRegisterCmd.Flags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
  tschRegisterCmd.Flags().StringVarP(&tschName, "name", "n", "", "Target task name")
  tschRegisterCmd.Flags().DurationVar(&tschStopDelay, "delay-stop", time.Duration(5*time.Second), "Delay between task execution and termination. This will not stop the process spawned by the task")
  tschRegisterCmd.Flags().DurationVarP(&tschDelay, "delay-start", "d", time.Duration(5*time.Second), "Delay between task registration and execution")
  tschRegisterCmd.Flags().DurationVarP(&tschDeleteDelay, "delay-delete", "D", time.Duration(0*time.Second), "Delay between task termination and deletion")
  tschRegisterCmd.Flags().BoolVar(&tschNoDelete, "no-delete", false, "Don't delete task after execution")
  tschRegisterCmd.Flags().BoolVar(&tschCallDelete, "call-delete", false, "Directly call SchRpcDelete to delete task")

  tschRegisterCmd.MarkFlagsMutuallyExclusive("no-delete", "delay-delete")
  tschRegisterCmd.MarkFlagsMutuallyExclusive("no-delete", "call-delete")
  tschRegisterCmd.MarkFlagsMutuallyExclusive("delay-delete", "call-delete")

  if err := tschRegisterCmd.MarkFlagRequired("executable"); err != nil {
    panic(err)
  }
}

var (
  tschNoDelete    bool
  tschCallDelete  bool
  tschDeleteDelay time.Duration
  tschStopDelay   time.Duration
  tschDelay       time.Duration
  tschName        string
  tschTaskPath    string

  tschCmd = &cobra.Command{
    Use:   "tsch",
    Short: "Establish execution via TSCH (ITaskSchedulerService)",
    Args:  cobra.NoArgs,
  }
  tschRegisterCmd = &cobra.Command{
    Use:   "register [target]",
    Short: "Register a remote scheduled task with an automatic start time",
    Long: `Description:
  The register method calls SchRpcRegisterTask to register a scheduled task
  with an automatic start time.This method avoids directly calling SchRpcRun,
  and can even avoid calling SchRpcDelete by populating the DeleteExpiredTaskAfter
  Setting.

References:
  SchRpcRegisterTask - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167
  SchRpcRun - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/77f2250d-500a-40ee-be18-c82f7079c4f0
  SchRpcDelete - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/360bb9b1-dd2a-4b36-83ee-21f12cb97cff
  DeleteExpiredTaskAfter - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/6bfde6fe-440e-4ddd-b4d6-c8fc0bc06fae
`,
    Args: needsRpcTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {

      log = log.With().
        Str("module", "tsch").
        Str("method", "register").
        Logger()
      if tschNoDelete {
        log.Warn().Msg("Task will not be deleted after execution")
      }

      module := tschexec.Module{}
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod: exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dce.ConnectionMethodDCEConfig{
          Endpoint: dceStringBinding,
          Options:  dceOptions,
          EpmAuto:  argDceEpmAuto,
        },
      }
      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: tschexec.MethodRegister,

        ExecutionMethodConfig: tschexec.MethodRegisterConfig{
          NoDelete:    tschNoDelete,
          CallDelete:  tschCallDelete,
          StartDelay:  tschDelay,
          StopDelay:   tschStopDelay,
          DeleteDelay: tschDeleteDelay,
          TaskPath:    tschTaskPath,
        },
      }
      if err := module.Connect(log.WithContext(ctx), creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      } else if err = module.Exec(log.WithContext(ctx), execCfg); err != nil {
        log.Fatal().Err(err).Msg("Execution failed")
      }
    },
  }
  tschDemandCmd = &cobra.Command{
    Use:   "demand [target]",
    Short: "Register a remote scheduled task and demand immediate start",
    Long: `Description:
  Similar to the register method, the demand method will call SchRpcRegisterTask,
  But rather than setting a defined time when the task will start, it will
  additionally call SchRpcRun to forcefully start the task.

References:
  SchRpcRegisterTask - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167
  SchRpcRun - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/77f2250d-500a-40ee-be18-c82f7079c4f0
`,
    Args: needsTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {

      log = log.With().
        Str("module", "tsch").
        Str("method", "register").
        Logger()
      if tschNoDelete {
        log.Warn().Msg("Task will not be deleted after execution")
      }
      module := tschexec.Module{}
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod: exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dce.ConnectionMethodDCEConfig{
          Endpoint: dceStringBinding,
          Options:  dceOptions,
          EpmAuto:  argDceEpmAuto,
        },
      }
      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: tschexec.MethodDemand,

        ExecutionMethodConfig: tschexec.MethodDemandConfig{
          NoDelete: tschNoDelete,
          TaskName: tschName,
        },
      }
      if err := module.Connect(log.WithContext(ctx), creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      } else if err = module.Exec(log.WithContext(ctx), execCfg); err != nil {
        log.Fatal().Err(err).Msg("Execution failed")
      }
    },
  }
  tschDeleteCmd = &cobra.Command{
    Use:   "delete [target]",
    Short: "Manually delete a scheduled task",
    Long: `Description:
  The delete method manually deletes a scheduled task by calling SchRpcDelete

References:
  SchRpcDelete - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/360bb9b1-dd2a-4b36-83ee-21f12cb97cff
`,
    Args: needsTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {
      log = log.With().
        Str("module", "tsch").
        Str("method", "delete").
        Logger()

      module := tschexec.Module{}
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod: exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dce.ConnectionMethodDCEConfig{
          Endpoint: dceStringBinding,
          Options:  dceOptions,
          EpmAuto:  argDceEpmAuto,
        },
      }
      cleanCfg := &exec.CleanupConfig{
        CleanupMethod:       tschexec.MethodDelete,
        CleanupMethodConfig: tschexec.MethodDeleteConfig{TaskPath: tschTaskPath},
      }
      if err := module.Connect(log.WithContext(ctx), creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      } else if err := module.Cleanup(log.WithContext(ctx), cleanCfg); err != nil {
        log.Fatal().Err(err).Msg("Cleanup failed")
      }
    },
  }
)
