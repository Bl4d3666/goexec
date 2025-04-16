package cmd

import (
  "context"
  dcomexec "github.com/FalconOpsLLC/goexec/pkg/goexec/dcom"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/spf13/cobra"
)

func dcomCmdInit() {
  registerRpcFlags(dcomCmd)
  dcomMmcCmdInit()
  dcomCmd.AddCommand(dcomMmcCmd)
}

func dcomMmcCmdInit() {
  dcomMmcCmd.Flags().StringVarP(&dcomMmc.WorkingDirectory, "directory", "d", `C:\`, "Working directory")
  dcomMmcCmd.Flags().StringVar(&dcomMmc.WindowState, "window", "Minimized", "Window state")

  registerProcessExecutionArgs(dcomMmcCmd)
}

var (
  dcomMmc dcomexec.DcomMmc

  dcomCmd = &cobra.Command{
    Use:   "dcom",
    Short: "Establish execution via DCOM",
    Args:  cobra.NoArgs,
  }
  dcomMmcCmd = &cobra.Command{
    Use:   "mmc [target]",
    Short: "Establish execution via the DCOM MMC20.Application object",
    Long: `Description:
  The mmc method uses the exposed MMC20.Application object to call Document.ActiveView.ShellExec,
  and ultimately execute system commands.

References:
  https://www.scorpiones.io/articles/lateral-movement-using-dcom-objects
  https://enigma0x3.net/2017/01/05/lateral-movement-using-the-mmc20-application-com-object/
  https://github.com/fortra/impacket/blob/master/examples/dcomexec.py
  https://learn.microsoft.com/en-us/previous-versions/windows/desktop/mmc/view-executeshellcommand
`,
    Args: argsRpcClient("host"),
    Run: func(cmd *cobra.Command, args []string) {
      var err error

      ctx := gssapi.NewSecurityContext(context.Background())

      ctx = log.With().
        Str("module", "dcom").
        Str("method", "mmc").
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

      if err = dcomMmc.Init(ctx, &rpcClient); err != nil {
        log.Error().Err(err).Msg("Module initialization failed")
        returnCode = 1
        return
      }

      if err = dcomMmc.Execute(ctx, exec.Input); err != nil {
        log.Error().Err(err).Msg("Execution failed")
        returnCode = 1
      }
    },
  }
)
