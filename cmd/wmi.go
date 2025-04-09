package cmd

import (
  "encoding/json"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  wmiexec "github.com/FalconOpsLLC/goexec/internal/exec/wmi"
  "github.com/spf13/cobra"
)

func wmiCmdInit() {
  registerRpcFlags(wmiCmd)
  wmiCallCmdInit()
  wmiCmd.AddCommand(wmiCallCmd)
  wmiProcessCmdInit()
  wmiCmd.AddCommand(wmiProcessCmd)
}

func wmiCallCmdInit() {
  wmiCallCmd.Flags().StringVarP(&dceConfig.Resource, "namespace", "n", "//./root/cimv2", "WMI namespace")
  wmiCallCmd.Flags().StringVarP(&wmi.Class, "class", "C", "", `WMI class to instantiate (i.e. "Win32_Process")`)
  wmiCallCmd.Flags().StringVarP(&wmi.Method, "method", "m", "", `WMI Method to call (i.e. "Create")`)
  wmiCallCmd.Flags().StringVarP(&wmi.Args, "args", "A", "{}", `WMI Method argument(s) in JSON dictionary format (i.e. {"CommandLine":"calc.exe"})`)
  if err := wmiCallCmd.MarkFlagRequired("method"); err != nil {
    panic(err)
  }
}

func wmiProcessCmdInit() {
  wmiProcessCmd.Flags().StringVarP(&command, "command", "c", "", "Process command line")
  wmiProcessCmd.Flags().StringVarP(&workingDirectory, "directory", "d", `C:\`, "Working directory")
  if err := wmiProcessCmd.MarkFlagRequired("command"); err != nil {
    panic(err)
  }
}

var (
  wmi struct {
    Class  string
    Method string
    Args   string
  }
  wmiMethodArgsMap map[string]any

  wmiCmd = &cobra.Command{
    Use:   "wmi",
    Short: "Establish execution via WMI",
    Args:  cobra.NoArgs,
  }
  wmiCallCmd = &cobra.Command{
    Use:   "call",
    Short: "Execute specified WMI method",
    Long: `Description:
  The call method creates an instance of the specified WMI class (-c),
  then calls the provided method (-m) with the provided arguments (-A).

References:
  https://learn.microsoft.com/en-us/windows/win32/wmisdk/wmi-classes
  `,
    Args: needs(needsTarget("cifs"), needsRpcTarget("cifs"), func(cmd *cobra.Command, args []string) (err error) {
      if err = json.Unmarshal([]byte(wmi.Args), &wmiMethodArgsMap); err != nil {
        err = fmt.Errorf("parse JSON arguments: %w", err)
      }
      return
    }),
    Run: func(cmd *cobra.Command, args []string) {
      executor := wmiexec.Module{}
      cleanCfg := &exec.CleanupConfig{} // TODO
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod:       exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dceConfig,
      }

      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: wmiexec.MethodCall,
        ExecutionMethodConfig: wmiexec.MethodCallConfig{
          Class:     wmi.Class,
          Method:    wmi.Method,
          Arguments: wmiMethodArgsMap,
        },
      }

      ctx = log.With().
        Str("module", "wmi").
        Str("method", "proc").
        Logger().WithContext(ctx)

      if err := executor.Connect(ctx, creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }
      defer func() {
        if err := executor.Cleanup(ctx, cleanCfg); err != nil {
          log.Error().Err(err).Msg("Cleanup failed")
        }
      }()
      if err := executor.Exec(ctx, execCfg); err != nil {
        log.Error().Err(err).Msg("Execution failed")
      }
    },
  }

  wmiProcessCmd = &cobra.Command{
    Use:   "proc",
    Short: "Start a Windows process",
    Long: `Description:
  The proc method creates an instance of the Win32_Process WMI class, then
  calls the Win32_Process.Create method with the provided command (-c),
  and optional working directory (-d).

References:
  https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/create-method-in-class-win32-process
`,
    Args: needs(needsTarget("cifs"), needsRpcTarget("cifs")),
    Run: func(cmd *cobra.Command, args []string) {

      executor := wmiexec.Module{}
      cleanCfg := &exec.CleanupConfig{} // TODO
      connCfg := &exec.ConnectionConfig{
        ConnectionMethod:       exec.ConnectionMethodDCE,
        ConnectionMethodConfig: dceConfig,
      }
      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: wmiexec.MethodProcess,

        ExecutionMethodConfig: wmiexec.MethodProcessConfig{
          Command:          command,
          WorkingDirectory: workingDirectory,
        },
      }

      ctx = log.With().
        Str("module", "wmi").
        Str("method", "proc").
        Logger().WithContext(ctx)

      if err := executor.Connect(ctx, creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")
      }
      defer func() {
        if err := executor.Cleanup(ctx, cleanCfg); err != nil {
          log.Error().Err(err).Msg("Cleanup failed")
        }
      }()
      if err := executor.Exec(ctx, execCfg); err != nil {
        log.Error().Err(err).Msg("Execution failed")
      }
    },
  }
)
