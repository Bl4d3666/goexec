package cmd

import (
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/FalconOpsLLC/goexec/internal/windows"
  "github.com/RedTeamPentesting/adauth"
  "github.com/spf13/cobra"

  scmrexec "github.com/FalconOpsLLC/goexec/internal/exec/scmr"
)

func scmrCmdInit() {
  registerRpcFlags(scmrCmd)
  scmrCmd.PersistentFlags().StringVarP(&executablePath, "executable-path", "f", "", "Full path to remote Windows executable")
  scmrCmd.PersistentFlags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
  scmrCmd.PersistentFlags().StringVarP(&scmrName, "service-name", "s", "", "Name of service to create or modify")

  scmrCmd.MarkPersistentFlagRequired("executable-path")
  scmrCmd.MarkPersistentFlagRequired("service-name")

  scmrCmd.AddCommand(scmrChangeCmd)
  scmrCreateCmdInit()
  scmrCmd.AddCommand(scmrCreateCmd)
  scmrChangeCmdInit()
}

func scmrChangeCmdInit() {
  scmrChangeCmd.Flags().StringVarP(&scmrDisplayName, "display-name", "n", "", "Display name of service to create")
  scmrChangeCmd.Flags().BoolVar(&scmrNoStart, "no-start", false, "Don't start service")
}

func scmrCreateCmdInit() {
  scmrCreateCmd.Flags().BoolVar(&scmrNoDelete, "no-delete", false, "Don't delete service after execution")
}

var (
  // scmr arguments
  scmrName        string
  scmrDisplayName string
  scmrNoDelete    bool
  scmrNoStart     bool

  scmrArgs = func(cmd *cobra.Command, args []string) (err error) {
    if len(args) != 1 {
      return fmt.Errorf("expected exactly 1 positional argument, got %d", len(args))
    }
    if creds, target, err = authOpts.WithTarget(ctx, "cifs", args[0]); err != nil {
      return fmt.Errorf("failed to parse target: %w", err)
    }
    log.Debug().Str("target", args[0]).Msg("Resolved target")
    return nil
  }

  creds  *adauth.Credential
  target *adauth.Target

  scmrCmd = &cobra.Command{
    Use:   "scmr",
    Short: "Establish execution via SCMR",
    Args:  cobra.NoArgs,
  }
  scmrCreateCmd = &cobra.Command{
    Use:   "create [target]",
    Short: "Create & run a new Windows service to gain execution",
    Args:  scmrArgs,
    RunE: func(cmd *cobra.Command, args []string) (err error) {
      if scmrNoDelete {
        log.Warn().Msg("Service will not be deleted after execution")
      }
      if scmrDisplayName == "" {
        scmrDisplayName = scmrName
        log.Warn().Msg("No display name specified, using service name as display name")
      }
      executor := scmrexec.Module{}
      execCfg := &exec.ExecutionConfig{
        ExecutablePath:  executablePath,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: scmrexec.MethodCreate,

        ExecutionMethodConfig: scmrexec.MethodCreateConfig{
          NoDelete:    scmrNoDelete,
          ServiceName: scmrName,
          DisplayName: scmrDisplayName,
          ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
          StartType:   windows.SERVICE_DEMAND_START,
        },
      }
      if err := executor.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
        log.Fatal().Err(err).Msg("SCMR execution failed")
      }
      return nil
    },
  }
  scmrChangeCmd = &cobra.Command{
    Use:   "change [target]",
    Short: "Change an existing Windows service to gain execution",
    Args:  scmrArgs,
    Run: func(cmd *cobra.Command, args []string) {
      executor := scmrexec.Module{}
      execCfg := &exec.ExecutionConfig{
        ExecutablePath:  executablePath,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: scmrexec.MethodModify,

        ExecutionMethodConfig: scmrexec.MethodModifyConfig{
          NoStart:     scmrNoStart,
          ServiceName: scmrName,
        },
      }
      if err := executor.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
        log.Fatal().Err(err).Msg("SCMR execution failed")
      }
    },
  }
)
