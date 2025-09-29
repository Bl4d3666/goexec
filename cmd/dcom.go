package cmd

import (
  "context"
  "fmt"
  "io"
  "os"

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
  dcomHtafileCmdInit()
  dcomExcelXlmCmdInit()

  dcomCmd.PersistentFlags().AddFlagSet(defaultAuthFlags.Flags)
  dcomCmd.PersistentFlags().AddFlagSet(defaultLogFlags.Flags)
  dcomCmd.PersistentFlags().AddFlagSet(defaultNetRpcFlags.Flags)
  dcomCmd.AddCommand(
    dcomMmcCmd,
    dcomShellWindowsCmd,
    dcomShellBrowserWindowCmd,
    dcomHtafileCmd,
    dcomExcelXlmCmd,
  )
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
  dcomShellWindowsExecFlags.Flags.StringVar(&dcomShellWindows.WorkingDirectory, "directory", `C:\`, "Working directory `path`")
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
  dcomShellBrowserWindowExecFlags.Flags.StringVar(&dcomShellBrowserWindow.WorkingDirectory, "directory", `C:\`, "Working directory `path`")
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

func dcomHtafileCmdInit() {
  dcomHtafileExecFlags := newFlagSet("Execution")
  dcomHtafileExecFlags.Flags.StringVar(&dcomHtafile.Url, "url", "", "Load custom `URL`")
  registerExecutionFlags(dcomHtafileExecFlags.Flags)
  registerExecutionOutputFlags(dcomHtafileExecFlags.Flags)

  cmdFlags[dcomHtafileCmd] = []*flagSet{
    dcomHtafileExecFlags,
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }
  dcomHtafileCmd.Flags().AddFlagSet(dcomHtafileExecFlags.Flags)

  // Constraints
  dcomHtafileCmd.MarkFlagsOneRequired("command", "exec", "url")
}

func dcomExcelXlmCmdInit() {
  dcomExcelXlmExecFlags := newFlagSet("Execution")
  dcomExcelXlmExecFlags.Flags.StringVarP(&dcomExcelXlm.Macro, "macro", "M", "", "XLM macro")
  dcomExcelXlmExecFlags.Flags.StringVar(&dcomExcelXlm.MacroFile, "macro-file", "", "XLM macro `file`")
  registerExecutionFlags(dcomExcelXlmExecFlags.Flags)
  registerExecutionOutputFlags(dcomExcelXlmExecFlags.Flags)

  cmdFlags[dcomExcelXlmCmd] = []*flagSet{
    dcomExcelXlmExecFlags,
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }
  dcomExcelXlmCmd.Flags().AddFlagSet(dcomExcelXlmExecFlags.Flags)

  // Constraints
  dcomExcelXlmCmd.MarkFlagsOneRequired("command", "exec", "macro", "macro-file")
  dcomExcelXlmCmd.MarkFlagsMutuallyExclusive("macro", "macro-file", "out")
}

var (
  dcomMmc                = dcomexec.DcomMmc{}
  dcomShellWindows       = dcomexec.DcomShellWindows{}
  dcomShellBrowserWindow = dcomexec.DcomShellBrowserWindow{}
  dcomHtafile            = dcomexec.DcomHtafile{}
  dcomExcelXlm           = dcomexec.DcomExcelXlm{}

  dcomCmd = &cobra.Command{
    Use:   "dcom",
    Short: "Execute with Distributed Component Object Model (MS-DCOM)",
    Long: `Description:
  The dcom module uses exposed Distributed Component Object Model (DCOM) objects to spawn processes.`,
    GroupID: "module",
    Args:    cobra.ArbitraryArgs,
  }

  dcomMmcCmd = &cobra.Command{
    Use:   "mmc [target]",
    Short: "Execute with the MMC20.Application DCOM object",
    Long: `Description:
  The mmc method uses the exposed MMC20.Application object to call Document.ActiveView.ShellExec,
  and ultimately spawn a process on the remote host.`,
    Args: args(argsRpcClient("host", ""),
      argsOutput("smb"),
      argsAcceptValues("window", &dcomMmc.WindowState, "Minimized", "Maximized", "Restored"),
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomMmc.Client = &rpcClient
      ctx := log.With().Str("module", dcomexec.ModuleName).Str("method", dcomexec.MethodMmc).
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
    Args: args(argsRpcClient("host", ""),
      argsOutput("smb"),
      argsAcceptValues("app-window", &dcomShellWindows.WindowState, "0", "1", "2", "3", "4", "5", "7", "10"),
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomShellWindows.Client = &rpcClient
      ctx := log.With().Str("module", dcomexec.ModuleName).Str("method", dcomexec.MethodShellWindows).
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
    Args: args(argsRpcClient("host", ""),
      argsOutput("smb"),
      argsAcceptValues("app-window", &dcomShellBrowserWindow.WindowState, "0", "1", "2", "3", "4", "5", "7", "10"),
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomShellBrowserWindow.Client = &rpcClient
      ctx := log.With().Str("module", dcomexec.ModuleName).Str("method", dcomexec.MethodShellBrowserWindow).
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteCleanMethod(ctx, &dcomShellBrowserWindow, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }

  dcomHtafileCmd = &cobra.Command{
    Use:   "htafile [target]",
    Short: "Execute with the HTAFile DCOM object",
    Long: `Description:
  The htafile method uses the exposed "HTML Application" DCOM object to load a remote HTA application or execute inline.
  This is made possible by the Load method of the IPersistMoniker interface.`,
    Args: args(argsRpcClient("host", ""),
      argsOutput("smb"),
      func(*cobra.Command, []string) error {
        return dcomexec.CheckUrlLength(dcomHtafile.Url, &exec)
      },
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomHtafile.Client = &rpcClient
      ctx := log.With().Str("module", dcomexec.ModuleName).Str("method", dcomexec.MethodHtafile).
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteCleanMethod(ctx, &dcomHtafile, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }

  dcomExcelXlmCmd = &cobra.Command{
    Use:   "excel-xlm [target]",
    Short: "Execute with the Excel.Application DCOM object using XLM macros",
    Long: `Description:
  The excel-xlm method uses the exposed Excel.Application DCOM object to call ExecuteExcel4Macro, thus executing
  XLM macros at will.`,
    Args: args(argsRpcClient("host", ""), argsOutput("smb"),
      func(*cobra.Command, []string) error {
        if dcomExcelXlm.MacroFile != "" {
          f, err := os.Open(dcomExcelXlm.MacroFile)
          if err != nil {
            return fmt.Errorf("open macro file: %w", err)
          }
          defer func() { _ = f.Close() }()
          b, err := io.ReadAll(f)
          if err != nil {
            return fmt.Errorf("read macro file: %w", err)
          }
          dcomExcelXlm.Macro = string(b)
        }
        return nil
      },
    ),
    Run: func(cmd *cobra.Command, args []string) {
      dcomExcelXlm.Client = &rpcClient
      ctx := log.With().Str("module", dcomexec.ModuleName).Str("method", dcomexec.MethodExcelXlm).
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteCleanMethod(ctx, &dcomExcelXlm, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }
)
