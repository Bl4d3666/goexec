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
	dcomShellWindowsCmdInit()
	dcomShellBrowserWindowCmdInit()

	dcomCmd.PersistentFlags().AddFlagSet(defaultAuthFlags.Flags)
	dcomCmd.PersistentFlags().AddFlagSet(defaultLogFlags.Flags)
	dcomCmd.PersistentFlags().AddFlagSet(defaultNetRpcFlags.Flags)
	dcomCmd.AddCommand(dcomMmcCmd, dcomShellWindowsCmd, dcomShellBrowserWindowCmd)
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

	// Constraints
	dcomMmcCmd.MarkFlagsOneRequired("command", "exec")
}

func dcomShellWindowsCmdInit() {
	dcomShellWindowsExecFlags := newFlagSet("Execution")

	registerExecutionFlags(dcomShellWindowsExecFlags.Flags)
	registerExecutionOutputFlags(dcomShellWindowsExecFlags.Flags)

	dcomShellWindowsExecFlags.Flags.StringVar(&dcomShellWindows.WorkingDirectory, "directory", `C:\`, "Working `directory`")
	dcomShellWindowsExecFlags.Flags.StringVar(&dcomShellWindows.WindowState, "app-window", "0", "Application window state `ID`")

	cmdFlags[dcomShellWindowsCmd] = []*flagSet{
		dcomShellWindowsExecFlags,
		defaultAuthFlags,
		defaultLogFlags,
		defaultNetRpcFlags,
	}
	dcomShellWindowsCmd.Flags().AddFlagSet(dcomShellWindowsExecFlags.Flags)

	// Constraints
	dcomShellWindowsCmd.MarkFlagsOneRequired("command", "exec")
}

func dcomShellBrowserWindowCmdInit() {
	dcomShellBrowserWindowExecFlags := newFlagSet("Execution")

	registerExecutionFlags(dcomShellBrowserWindowExecFlags.Flags)
	registerExecutionOutputFlags(dcomShellBrowserWindowExecFlags.Flags)

	dcomShellBrowserWindowExecFlags.Flags.StringVar(&dcomShellBrowserWindow.WorkingDirectory, "directory", `C:\`, "Working `directory`")
	dcomShellBrowserWindowExecFlags.Flags.StringVar(&dcomShellBrowserWindow.WindowState, "app-window", "0", "Application window state `ID`")

	cmdFlags[dcomShellBrowserWindowCmd] = []*flagSet{
		dcomShellBrowserWindowExecFlags,
		defaultAuthFlags,
		defaultLogFlags,
		defaultNetRpcFlags,
	}
	dcomShellBrowserWindowCmd.Flags().AddFlagSet(dcomShellBrowserWindowExecFlags.Flags)

	// Constraints
	dcomShellBrowserWindowCmd.MarkFlagsOneRequired("command", "exec")
}

var (
	dcomMmc                dcomexec.DcomMmc
	dcomShellWindows       dcomexec.DcomShellWindows
	dcomShellBrowserWindow dcomexec.DcomShellBrowserWindow

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
		Short: "Execute with the MMC20.Application DCOM object",
		Long: `Description:
  The mmc method uses the exposed MMC20.Application object to call Document.ActiveView.ShellExec,
  and ultimately spawn a process on the remote host.`,
		Args: args(
			argsRpcClient("host"),
			argsOutput("smb"),
			argsAcceptValues("window", &dcomMmc.WindowState, "Minimized", "Maximized", "Restored"),
		),
		Run: func(cmd *cobra.Command, args []string) {
			dcomMmc.Client = &rpcClient
			dcomMmc.IO = exec
			dcomMmc.ClassID = dcomexec.Mmc20Uuid

			ctx := log.With().
				Str("module", dcomexec.ModuleName).
				Str("method", dcomexec.MethodMmc).
				Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

			if err := goexec.ExecuteCleanMethod(ctx, &dcomMmc, &exec); err != nil {
				log.Fatal().Err(err).Msg("Operation failed")
			}
		},
	}

	dcomShellWindowsCmd = &cobra.Command{
		Use:   "shellwindows [target]",
		Short: "Execute with the ShellWindows DCOM object",
		Long: `Description:
  The shellwindows method uses the exposed ShellWindows DCOM object on older Windows installations
  to call Item().Document.Application.ShellExecute, and spawn the provided process.`,
		Args: args(
			argsRpcClient("host"),
			argsOutput("smb"),
			argsAcceptValues("app-window", &dcomShellWindows.WindowState, "0", "1", "2", "3", "4", "5", "7", "10"),
		),
		Run: func(cmd *cobra.Command, args []string) {
			dcomShellWindows.Client = &rpcClient
			dcomShellWindows.IO = exec
			dcomShellWindows.ClassID = dcomexec.ShellWindowsUuid

			ctx := log.With().
				Str("module", dcomexec.ModuleName).
				Str("method", dcomexec.MethodShellWindows).
				Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

			if err := goexec.ExecuteCleanMethod(ctx, &dcomShellWindows, &exec); err != nil {
				log.Fatal().Err(err).Msg("Operation failed")
			}
		},
	}

	dcomShellBrowserWindowCmd = &cobra.Command{
		Use:   "shellbrowserwindow [target]",
		Short: "Execute with the ShellBrowserWindow DCOM object",
		Long: `Description:
  The shellbrowserwindow method uses the exposed ShellBrowserWindow DCOM object on older Windows installations
  to call Document.Application.ShellExecute, and spawn the provided process.`,
		Args: args(
			argsRpcClient("host"),
			argsOutput("smb"),
			argsAcceptValues("app-window", &dcomShellBrowserWindow.WindowState, "0", "1", "2", "3", "4", "5", "7", "10"),
		),
		Run: func(cmd *cobra.Command, args []string) {
			dcomShellBrowserWindow.Client = &rpcClient
			dcomShellBrowserWindow.IO = exec
			dcomShellBrowserWindow.ClassID = dcomexec.ShellBrowserWindowUuid

			ctx := log.With().
				Str("module", dcomexec.ModuleName).
				Str("method", dcomexec.MethodShellBrowserWindow).
				Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

			if err := goexec.ExecuteCleanMethod(ctx, &dcomShellBrowserWindow, &exec); err != nil {
				log.Fatal().Err(err).Msg("Operation failed")
			}
		},
	}
)
