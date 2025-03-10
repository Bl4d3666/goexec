package cmd

import (
	"github.com/FalconOpsLLC/goexec/internal/exec"
	dcomexec "github.com/FalconOpsLLC/goexec/internal/exec/dcom"
	"github.com/spf13/cobra"
)

func dcomCmdInit() {
	registerRpcFlags(dcomCmd)
	dcomMmcCmdInit()
	dcomCmd.AddCommand(dcomMmcCmd)
}

func dcomMmcCmdInit() {
	dcomMmcCmd.Flags().StringVarP(&executable, "executable", "e", "", "Remote Windows executable to invoke")
	dcomMmcCmd.Flags().StringVarP(&workingDirectory, "directory", "d", `C:\`, "Working directory")
	dcomMmcCmd.Flags().StringVarP(&executableArgs, "args", "a", "", "Process command line")
	dcomMmcCmd.Flags().StringVar(&windowState, "window", "Minimized", "Window state")
	dcomMmcCmd.Flags().StringVarP(&command, "command", "c", ``, "Windows executable & arguments to run")

	dcomMmcCmd.MarkFlagsOneRequired("executable", "command")
	dcomMmcCmd.MarkFlagsMutuallyExclusive("executable", "command")
}

var (
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
		Args: needsRpcTarget("host"),
		Run: func(cmd *cobra.Command, args []string) {

			ctx = log.With().
				Str("module", "dcom").
				Str("method", "mmc").
				Logger().WithContext(ctx)

			module := dcomexec.Module{}
			connCfg := &exec.ConnectionConfig{
				ConnectionMethod:       exec.ConnectionMethodDCE,
				ConnectionMethodConfig: dceConfig,
			}
			execCfg := &exec.ExecutionConfig{
				ExecutableName:  executable,
				ExecutableArgs:  executableArgs,
				ExecutionMethod: dcomexec.MethodMmc,

				ExecutionMethodConfig: dcomexec.MethodMmcConfig{
					WorkingDirectory: workingDirectory,
					WindowState:      windowState,
				},
			}
			if err := module.Connect(ctx, creds, target, connCfg); err != nil {
				log.Fatal().Err(err).Msg("Connection failed")
			} else if err = module.Exec(ctx, execCfg); err != nil {
				log.Fatal().Err(err).Msg("Execution failed")
			}
		},
	}
)
