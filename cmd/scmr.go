package cmd

import (
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/FalconOpsLLC/goexec/internal/windows"
  "github.com/RedTeamPentesting/adauth"
  "github.com/spf13/cobra"

  scmrexec "github.com/FalconOpsLLC/goexec/internal/exec/scmr"
)

func scmrCmdInit() {
  registerRpcFlags(scmrCmd)
  scmrCmd.PersistentFlags().StringVarP(&executablePath, "executable-path", "f", "", "Full path to remote Windows executable")
  scmrCmd.PersistentFlags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
  scmrCmd.PersistentFlags().StringVarP(&scmrServiceName, "service-name", "s", "", "Name of service to create or modify")

  if err := scmrCmd.MarkPersistentFlagRequired("executable-path"); err != nil {
    panic(err)
  }
  scmrCreateCmdInit()
  scmrCmd.AddCommand(scmrChangeCmd)
  scmrChangeCmdInit()
  scmrCmd.AddCommand(scmrCreateCmd)
  scmrDeleteCmdInit()
  scmrCmd.AddCommand(scmrDeleteCmd)
}

func scmrCreateCmdInit() {
  scmrCreateCmd.Flags().StringVarP(&scmrServiceName, "service-name", "s", "", "Name of service to create")
  scmrCreateCmd.Flags().BoolVar(&scmrNoDelete, "no-delete", false, "Don't delete service after execution")
}

func scmrChangeCmdInit() {
  scmrChangeCmd.Flags().StringVarP(&scmrDisplayName, "display-name", "n", "", "Display name of service to create")
  scmrChangeCmd.Flags().BoolVar(&scmrNoStart, "no-start", false, "Don't start service")
  scmrChangeCmd.Flags().StringVarP(&scmrServiceName, "service-name", "s", "", "Name of service to modify")
  if err := scmrChangeCmd.MarkFlagRequired("service-name"); err != nil {
    panic(err)
  }
}

func scmrDeleteCmdInit() {
  scmrDeleteCmd.Flags().StringVarP(&scmrServiceName, "service-name", "s", "", "Name of service to delete")
  if err := scmrChangeCmd.MarkFlagRequired("service-name"); err != nil {
    panic(err)
  }
}

var (
  // scmr arguments
  scmrServiceName string
  scmrDisplayName string
  scmrNoDelete    bool
  scmrNoStart     bool

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
    Long: `Description:
  The create method calls RCreateServiceW to create a new Windows service with
  the provided executable & arguments as the lpBinaryPathName

References:
  https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/6a8ca926-9477-4dd4-b766-692fab07227e
`,
    Args: needsRpcTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {

      if scmrServiceName == "" {
        log.Warn().Msg("No service name was specified, using random string")
        scmrServiceName = util.RandomString()
      }
      if scmrNoDelete {
        log.Warn().Msg("Service will not be deleted after execution")
      }
      if scmrDisplayName == "" {
        log.Debug().Msg("No display name specified, using service name as display name")
        scmrDisplayName = scmrServiceName
      }

      executor := scmrexec.Module{}
      cleanCfg := &exec.CleanupConfig{
        CleanupMethod: scmrexec.CleanupMethodDelete,
      }
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod:       exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dceConfig,
      }
      execCfg := &exec.ExecutionConfig{
        ExecutablePath:  executablePath,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: scmrexec.MethodCreate,

        ExecutionMethodConfig: scmrexec.MethodCreateConfig{
          NoDelete:    scmrNoDelete,
          ServiceName: util.RandomStringIfBlank(scmrServiceName),
          DisplayName: scmrDisplayName,
          ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
          StartType:   windows.SERVICE_DEMAND_START,
        },
      }
      ctx = log.With().
        Str("module", "scmr").
        Str("method", "create").
        Logger().WithContext(ctx)

      if err := executor.Connect(ctx, creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }
      if !scmrNoDelete {
        defer func() {
          if err := executor.Cleanup(ctx, cleanCfg); err != nil {
            log.Error().Err(err).Msg("Cleanup failed")
          }
        }()
      }
      if err := executor.Exec(ctx, execCfg); err != nil {
        log.Error().Err(err).Msg("Execution failed")
      }
    },
  }
  scmrChangeCmd = &cobra.Command{
    Use:   "change [target]",
    Short: "Change an existing Windows service to gain execution",
    Args:  needsRpcTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {

      executor := scmrexec.Module{}
      cleanCfg := &exec.CleanupConfig{
        CleanupMethod: scmrexec.CleanupMethodRevert,
      }
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod:       exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dceConfig,
      }
      execCfg := &exec.ExecutionConfig{
        ExecutablePath:  executablePath,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: scmrexec.MethodChange,

        ExecutionMethodConfig: scmrexec.MethodChangeConfig{
          NoStart:     scmrNoStart,
          ServiceName: scmrServiceName,
        },
      }
      ctx = log.With().
        Str("module", "scmr").
        Str("method", "change").
        Logger().WithContext(ctx)

      if err := executor.Connect(ctx, creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }
      if !scmrNoDelete {
        defer func() {
          if err := executor.Cleanup(ctx, cleanCfg); err != nil {
            log.Error().Err(err).Msg("Cleanup failed")
          }
        }()
      }
      if err := executor.Exec(ctx, execCfg); err != nil {
        log.Error().Err(err).Msg("Execution failed")
      }
    },
  }
  scmrDeleteCmd = &cobra.Command{
    Use:   "delete [target]",
    Short: "Delete an existing Windows service",
    Long: `Description:
  
`,
    Args: needsRpcTarget("cifs"),
    Run: func(cmd *cobra.Command, args []string) {

      executor := scmrexec.Module{}
      cleanCfg := &exec.CleanupConfig{
        CleanupMethod: scmrexec.CleanupMethodDelete,
      }
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod:       exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dceConfig,
      }
      ctx = log.With().
        Str("module", "scmr").
        Str("method", "delete").
        Logger().WithContext(ctx)

      if err := executor.Connect(ctx, creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")

      } else if err = executor.Cleanup(ctx, cleanCfg); err != nil {
        log.Fatal().Err(err).Msg("Delete failed")
      }
    },
  }
)
