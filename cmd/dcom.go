package cmd

import (
  "context"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  dcomexec "github.com/FalconOpsLLC/goexec/pkg/goexec/dcom"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/spf13/cobra"
)

func dcomCmdInit() {
  cmdFlags[dcomCmd] = []*flagSet{
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }
  dcomMmcCmdInit()

  dcomCmd.PersistentFlags().AddFlagSet(defaultAuthFlags.Flags)
  dcomCmd.PersistentFlags().AddFlagSet(defaultLogFlags.Flags)
  dcomCmd.PersistentFlags().AddFlagSet(defaultNetRpcFlags.Flags)
  dcomCmd.AddCommand(dcomMmcCmd)
}

func dcomMmcCmdInit() {
  dcomMmcExecFlags := newFlagSet("Execution")

  registerExecutionFlags(dcomMmcExecFlags.Flags)
  registerExecutionOutputFlags(dcomMmcExecFlags.Flags)

  dcomMmcExecFlags.Flags.StringVar(&dcomMmc.WorkingDirectory, "directory", `C:\`, "Working `directory`")
  dcomMmcExecFlags.Flags.StringVar(&dcomMmc.WindowState, "window", "Minimized", "Window state")

  cmdFlags[dcomMmcCmd] = []*flagSet{
    dcomMmcExecFlags,
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }

  dcomMmcCmd.Flags().AddFlagSet(dcomMmcExecFlags.Flags)
}

var (
  dcomMmc dcomexec.DcomMmc

  dcomCmd = &cobra.Command{
    Use:   "dcom",
    Short: "Execute with Distributed Component Object Model (MS-DCOM)",
    Long: `Description:
  The dcom module uses exposed Distributed Component Object Model (DCOM) objects to spawn processes.`,
    GroupID: "module",
    Args:    cobra.NoArgs,
  }

  dcomMmcCmd = &cobra.Command{
    Use:   "mmc [target]",
    Short: "Execute with the DCOM MMC20.Application object",
    Long: `Description:
  The mmc method uses the exposed MMC20.Application object to call Document.ActiveView.ShellExec,
  and ultimately spawn a process on the remote host.`,
    Args: args(
      argsRpcClient("host"),
      argsOutput("smb"),
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomMmc.Client = &rpcClient
      dcomMmc.IO = exec

      ctx := log.With().
        Str("module", "dcom").
        Str("method", "mmc").
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteCleanMethod(ctx, &dcomMmc, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }
)
