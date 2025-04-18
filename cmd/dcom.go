package cmd

import (
  "context"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
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
  registerExecutionOutputArgs(dcomMmcCmd)
}

var (
  dcomMmc dcomexec.DcomMmc

  dcomCmd = &cobra.Command{
    Use:     "dcom",
    Short:   "Execute with Distributed Component Object Model (MS-DCOM)",
    GroupID: "module",
    Args:    cobra.NoArgs,
  }

  dcomMmcCmd = &cobra.Command{
    Use:   "mmc [target]",
    Short: "Execute with the DCOM MMC20.Application object",
    Long: `Description:
  The mmc method uses the exposed MMC20.Application object to call Document.ActiveView.ShellExec,
  and ultimately spawn a process on the remote host.

References:
  - https://www.scorpiones.io/articles/lateral-movement-using-dcom-objects
  - https://enigma0x3.net/2017/01/05/lateral-movement-using-the-mmc20-application-com-object/
  - https://github.com/fortra/impacket/blob/master/examples/dcomexec.py
  - https://learn.microsoft.com/en-us/previous-versions/windows/desktop/mmc/view-executeshellcommand
`,
    Args: args(
      argsRpcClient("host"),
      argsOutput("smb"),
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomMmc.Dcom.Client = &rpcClient
      dcomMmc.IO = exec

      ctx := log.WithContext(gssapi.NewSecurityContext(context.TODO()))

      if err := goexec.ExecuteCleanMethod(ctx, &dcomMmc, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }
)
