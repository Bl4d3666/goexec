package cmd

import (
  "encoding/json"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  wmiexec "github.com/FalconOpsLLC/goexec/internal/exec/wmi"
  "github.com/spf13/cobra"
  "regexp"
  "strings"
)

func wmiCmdInit() {
  wmiCustomCmdInit()
  wmiCmd.AddCommand(wmiCustomCmd)
  wmiProcessCmdInit()
  wmiCmd.AddCommand(wmiProcessCmd)
}

func wmiCustomCmdInit() {
  wmiCustomCmd.Flags().StringVarP(&wmiArgMethod, "method", "m", "", `WMI Method to use in the format CLASS.METHOD (i.e. "Win32_Process.Create")`)
  wmiCustomCmd.Flags().StringVarP(&wmiArgMethodArgs, "args", "A", "{}", `WMI Method argument(s) in JSON dictionary format (i.e. {"CommandLine":"calc.exe"})`)
  if err := wmiCustomCmd.MarkFlagRequired("method"); err != nil {
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
  // for custom method
  wmiArgMethod     string
  wmiArgMethodArgs string

  wmiClass         string
  wmiMethod        string
  wmiMethodArgsMap map[string]any
  methodRegex      = regexp.MustCompile(`^\w+\.\w+$`)

  wmiCmd = &cobra.Command{
    Use:   "wmi",
    Short: "Establish execution via WMI",
    Args:  cobra.NoArgs,
  }
  wmiCustomCmd = &cobra.Command{
    Use:   "custom",
    Short: "Execute specified WMI method",
    Long: `Description:
  The custom method creates an instance of the specified WMI class (-c),
  then calls the provided method (-m) with the provided arguments (-A).

References:
  https://learn.microsoft.com/en-us/windows/win32/wmisdk/wmi-classes
  `,
    Args: func(cmd *cobra.Command, args []string) (err error) {
      if err = needsTarget(cmd, args); err == nil {
        if wmiArgMethod != "" && !methodRegex.MatchString(wmiArgMethod) {
          return fmt.Errorf("invalid CLASS.METHOD syntax: %s", wmiArgMethod)
        }
        if err = json.Unmarshal([]byte(wmiArgMethodArgs), &wmiMethodArgsMap); err != nil {
          err = fmt.Errorf("failed to parse JSON arguments: %w", err)
        }
      }
      return
    },
    Run: func(cmd *cobra.Command, args []string) {
      module := wmiexec.Module{}

      connCfg := &exec.ConnectionConfig{}
      cleanCfg := &exec.CleanupConfig{}

      parts := strings.SplitN(wmiArgMethod, ".", 2)
      wmiClass = parts[0]
      wmiMethod = parts[1]

      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: wmiexec.MethodCustom,
        ExecutionMethodConfig: wmiexec.MethodCustomConfig{
          Class:     wmiClass,
          Method:    wmiMethod,
          Arguments: wmiMethodArgsMap,
        },
      }
      if err := module.Connect(log.WithContext(ctx), creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")

      } else if err := module.Exec(log.WithContext(ctx), execCfg); err != nil {
        log.Fatal().Err(err).Msg("Execution failed")

      } else if err := module.Cleanup(log.WithContext(ctx), cleanCfg); err != nil {
        log.Error().Err(err).Msg("Cleanup failed")
      }
    },
  }

  wmiProcessCmd = &cobra.Command{
    Use:   "process",
    Short: "Create a Windows process",
    Long: `Description:
  The process method creates an instance of the Win32_Process WMI class,
  then calls the Win32_Process.Create method with the provided command (-c),
  and optional working directory (-d).

References:
  https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/create-method-in-class-win32-process
`,
    Args: needsTarget,
    Run: func(cmd *cobra.Command, args []string) {
      module := wmiexec.Module{}

      connCfg := &exec.ConnectionConfig{}
      cleanCfg := &exec.CleanupConfig{}

      execCfg := &exec.ExecutionConfig{
        ExecutableName:  executable,
        ExecutableArgs:  executableArgs,
        ExecutionMethod: wmiexec.MethodProcess,
        ExecutionMethodConfig: wmiexec.MethodProcessConfig{
          Command:          command,
          WorkingDirectory: workingDirectory,
        },
      }
      if err := module.Connect(log.WithContext(ctx), creds, target, connCfg); err != nil {
        log.Fatal().Err(err).Msg("Connection failed")

      } else if err := module.Exec(log.WithContext(ctx), execCfg); err != nil {
        log.Fatal().Err(err).Msg("Execution failed")

      } else if err := module.Cleanup(log.WithContext(ctx), cleanCfg); err != nil {
        log.Error().Err(err).Msg("Cleanup failed")
      }
    },
  }
)
