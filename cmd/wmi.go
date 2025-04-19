package cmd

import (
  "context"
  "encoding/json"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  wmiexec "github.com/FalconOpsLLC/goexec/pkg/goexec/wmi"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/spf13/cobra"
  "os"
)

func wmiCmdInit() {
  cmdFlags[wmiCmd] = []*flagSet{
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }
  wmiCallCmdInit()
  wmiProcCmdInit()

  wmiCmd.PersistentFlags().AddFlagSet(defaultAuthFlags.Flags)
  wmiCmd.PersistentFlags().AddFlagSet(defaultLogFlags.Flags)
  wmiCmd.PersistentFlags().AddFlagSet(defaultNetRpcFlags.Flags)
  wmiCmd.AddCommand(wmiProcCmd, wmiCallCmd)
}

func wmiCallCmdInit() {
  wmiCallFlags := newFlagSet("WMI")

  wmiCallFlags.Flags.StringVarP(&wmiCall.Resource, "namespace", "n", "//./root/cimv2", "WMI namespace")
  wmiCallFlags.Flags.StringVarP(&wmiCall.Class, "class", "C", "", `WMI class to instantiate (i.e. "Win32_Process")`)
  wmiCallFlags.Flags.StringVarP(&wmiCall.Method, "method", "m", "", `WMI Method to call (i.e. "Create")`)
  wmiCallFlags.Flags.StringVarP(&wmiArguments, "args", "A", "{}", `WMI Method argument(s) in JSON dictionary format (i.e. {"Command":"calc.exe"})`)

  wmiCallCmd.Flags().AddFlagSet(wmiCallFlags.Flags)

  cmdFlags[wmiCallCmd] = []*flagSet{
    wmiCallFlags,
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }
  if err := wmiCallCmd.MarkFlagRequired("class"); err != nil {
    panic(err)
  }
  if err := wmiCallCmd.MarkFlagRequired("method"); err != nil {
    panic(err)
  }
}

func wmiProcCmdInit() {
  wmiProcFlags := newFlagSet("WMI")

  wmiProcFlags.Flags.StringVarP(&wmiProc.Resource, "namespace", "n", "//./root/cimv2", "WMI namespace")
  wmiProcFlags.Flags.StringVarP(&wmiProc.WorkingDirectory, "directory", "d", `C:\`, "Working directory")

  wmiProcExecFlags := newFlagSet("Execution")

  registerExecutionFlags(wmiProcExecFlags.Flags)
  registerExecutionOutputFlags(wmiProcExecFlags.Flags)

  cmdFlags[wmiProcCmd] = []*flagSet{
    wmiProcExecFlags,
    wmiProcFlags,
    defaultAuthFlags,
    defaultLogFlags,
    defaultNetRpcFlags,
  }

  wmiProcCmd.Flags().AddFlagSet(wmiProcFlags.Flags)
  wmiProcCmd.Flags().AddFlagSet(wmiProcExecFlags.Flags)
}

var (
  wmiCall = wmiexec.WmiCall{}
  wmiProc = wmiexec.WmiProc{}

  wmiArguments string

  wmiCmd = &cobra.Command{
    Use:     "wmi",
    Short:   "Execute with Windows Management Instrumentation (MS-WMI)",
    GroupID: "module",
    Args:    cobra.NoArgs,
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
    Args: args(
      argsRpcClient("cifs"),
      func(cmd *cobra.Command, args []string) error {
        return json.Unmarshal([]byte(wmiArguments), &wmiCall.Args)
      }),

    Run: func(cmd *cobra.Command, args []string) {
      wmiCall.Client = &rpcClient
      wmiCall.Out = os.Stdout

      ctx := log.With().
        Str("module", "wmi").
        Str("method", "call").
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteAuxiliaryMethod(ctx, &wmiCall); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }

  wmiProcCmd = &cobra.Command{
    Use:   "proc",
    Short: "Start a Windows process",
    Long: `Description:
  The proc method creates an instance of the Win32_Process WMI class, then
  calls the Win32_Process.Create method with the provided command (-c),
  and optional working directory (-d).

References:
  https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/create-method-in-class-win32-process
`,
    Args: args(
      argsRpcClient("cifs"),
      argsOutput("smb"),
    ),

    Run: func(cmd *cobra.Command, args []string) {
      wmiProc.Client = &rpcClient
      wmiProc.IO = exec

      ctx := log.With().
        Str("module", "wmi").
        Str("method", "proc").
        Logger().WithContext(gssapi.NewSecurityContext(context.Background()))

      if err := goexec.ExecuteCleanMethod(ctx, &wmiProc, &exec); err != nil {
        log.Fatal().Err(err).Msg("Operation failed")
      }
    },
  }
)
