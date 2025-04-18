package cmd

import (
  "context"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/spf13/cobra"

  scmrexec "github.com/FalconOpsLLC/goexec/pkg/goexec/scmr"
)

func scmrCmdInit() {
  registerRpcFlags(scmrCmd)
  scmrCreateCmdInit()
  scmrCmd.AddCommand(scmrCreateCmd)
  scmrChangeCmdInit()
  scmrCmd.AddCommand(scmrChangeCmd)
  scmrDeleteCmdInit()
  scmrCmd.AddCommand(scmrDeleteCmd)
}

func scmrCreateCmdInit() {
  scmrCreateCmd.Flags().StringVarP(&scmrCreate.DisplayName, "display-name", "n", "", "Display name of service to create")
  scmrCreateCmd.Flags().StringVarP(&scmrCreate.ServiceName, "service-name", "s", "", "Name of service to create")
  scmrCreateCmd.Flags().BoolVar(&scmrCreate.NoDelete, "no-delete", false, "Don't delete service after execution")
  scmrCreateCmd.Flags().BoolVar(&scmrCreate.NoStart, "no-start", false, "Don't start service")

  scmrCreateCmd.Flags().StringVarP(&exec.Input.ExecutablePath, "executable-path", "f", "", "Full path to a remote Windows executable")
  scmrCreateCmd.Flags().StringVarP(&exec.Input.Arguments, "args", "a", "", "Arguments to pass to the executable")

  scmrCreateCmd.MarkFlagsMutuallyExclusive("no-delete", "no-start")

  if err := scmrCreateCmd.MarkFlagRequired("executable-path"); err != nil {
    panic(err)
  }
}

func scmrChangeCmdInit() {
  scmrChangeCmd.Flags().BoolVar(&scmrChange.NoStart, "no-start", false, "Don't start service")
  scmrChangeCmd.Flags().StringVarP(&scmrChange.ServiceName, "service-name", "s", "", "Name of service to modify")

  scmrChangeCmd.Flags().StringVarP(&exec.Input.ExecutablePath, "executable-path", "f", "", "Full path to remote Windows executable")
  scmrChangeCmd.Flags().StringVarP(&exec.Input.Arguments, "args", "a", "", "Arguments to pass to executable")

  if err := scmrChangeCmd.MarkFlagRequired("service-name"); err != nil {
    panic(err)
  }
  if err := scmrCreateCmd.MarkFlagRequired("executable-path"); err != nil {
    panic(err)
  }
}

func scmrDeleteCmdInit() {
  scmrDeleteCmd.Flags().StringVarP(&scmrDelete.ServiceName, "service-name", "s", scmrDelete.ServiceName, "Name of service to delete")

  if err := scmrDeleteCmd.MarkFlagRequired("service-name"); err != nil {
    panic(err)
  }
}

var (
  scmrCreate scmrexec.ScmrCreate
  scmrChange scmrexec.ScmrChange
  scmrDelete scmrexec.ScmrDelete

  scmrCmd = &cobra.Command{
    Use:     "scmr",
    Short:   "Execute with Service Control Manager Remote (MS-SCMR)",
    GroupID: "module",
    Args:    cobra.NoArgs,
  }

  scmrCreateCmd = &cobra.Command{
    Use:   "create [target]",
    Short: "Spawn a remote process by creating & running a Windows service",
    Long: `Description:
  The create method calls RCreateServiceW to create a new Windows service on the
  remote target with the provided executable & arguments as the lpBinaryPathName

References:
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/6a8ca926-9477-4dd4-b766-692fab07227e
`,
    Args: argsRpcClient("cifs"),

    Run: func(cmd *cobra.Command, args []string) {
      var err error

      ctx := gssapi.NewSecurityContext(context.Background())

      ctx = log.With().
        Str("module", "scmr").
        Str("method", "create").
        Logger().
        WithContext(ctx)

      if scmrCreate.ServiceName == "" {
        log.Warn().Msg("No service name was provided. Using a random string")
        scmrCreate.ServiceName = util.RandomString()
      }

      if scmrCreate.NoDelete {
        log.Warn().Msg("Service will not be deleted after execution")
      }

      if scmrCreate.DisplayName == "" {
        log.Debug().Msg("No display name specified, using service name as display name")
        scmrCreate.DisplayName = scmrCreate.ServiceName
      }

      if err = rpcClient.Connect(ctx); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }

      defer func() {
        closeErr := rpcClient.Close(ctx)
        if closeErr != nil {
          log.Error().Err(closeErr).Msg("Failed to close connection")
        }
      }()

      defer func() {
        cleanErr := scmrCreate.Clean(ctx)
        if cleanErr != nil {
          log.Warn().Err(cleanErr).Msg("Clean operation failed")
        }
      }()

      if err = scmrCreate.Init(ctx, &rpcClient); err != nil {
        log.Error().Err(err).Msg("Module initialization failed")
        returnCode = 2
        return
      }

      if err = scmrCreate.Execute(ctx, exec.Input); err != nil {
        log.Error().Err(err).Msg("Execution failed")
        returnCode = 4
      }
    },
  }

  scmrChangeCmd = &cobra.Command{
    Use:   "change [target]",
    Short: "Change an existing Windows service to gain execution",
    Args:  argsRpcClient("cifs"),
    Run: func(cmd *cobra.Command, args []string) {
      var err error

      ctx := gssapi.NewSecurityContext(context.Background())

      ctx = log.With().
        Str("module", "scmr").
        Str("method", "change").
        Logger().
        WithContext(ctx)

      if err = rpcClient.Connect(ctx); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }

      defer func() {
        closeErr := rpcClient.Close(ctx)
        if closeErr != nil {
          log.Error().Err(closeErr).Msg("Failed to close connection")
        }
      }()

      defer func() {
        cleanErr := scmrChange.Clean(ctx)
        if cleanErr != nil {
          log.Warn().Err(cleanErr).Msg("Clean operation failed")
        }
      }()

      if err = scmrChange.Init(ctx, &rpcClient); err != nil {
        log.Error().Err(err).Msg("Module initialization failed")
        returnCode = 2
        return
      }

      if err = scmrChange.Execute(ctx, exec.Input); err != nil {
        log.Error().Err(err).Msg("Execution failed")
        returnCode = 4
      }
    },
  }
  scmrDeleteCmd = &cobra.Command{
    Use:   "delete [target]",
    Short: "Delete an existing Windows service",
    Long:  `TODO`,

    Args: argsRpcClient("cifs"),
    Run: func(cmd *cobra.Command, args []string) {
      var err error

      ctx := gssapi.NewSecurityContext(context.Background())

      ctx = log.With().
        Str("module", "scmr").
        Str("method", "delete").
        Logger().
        WithContext(ctx)

      if err = rpcClient.Connect(ctx); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }

      defer func() {
        closeErr := rpcClient.Close(ctx)
        if closeErr != nil {
          log.Error().Err(closeErr).Msg("Failed to close connection")
        }
      }()

      if err = scmrDelete.Init(ctx, &rpcClient); err != nil {
        log.Error().Err(err).Msg("Module initialization failed")
        returnCode = 2
      }

      if err = scmrDelete.Clean(ctx); err != nil {
        log.Warn().Err(err).Msg("Clean failed")
        returnCode = 4
      }
    },
  }
)
