package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/FalconOpsLLC/goexec/pkg/goexec"
	wmiexec "github.com/FalconOpsLLC/goexec/pkg/goexec/wmi"
	"github.com/oiweiwei/go-msrpc/ssp/gssapi"
	"github.com/spf13/cobra"
)

func wmiCmdInit() {
	registerRpcFlags(wmiCmd)

	wmiCallCmdInit()
	wmiCmd.AddCommand(wmiCallCmd)

	wmiProcCmdInit()
	wmiCmd.AddCommand(wmiProcCmd)
}

func wmiCallArgs(_ *cobra.Command, _ []string) error {
	return json.Unmarshal([]byte(wmiArguments), &wmiCall.Args)
}

func wmiCallCmdInit() {
	wmiCallCmd.Flags().StringVarP(&wmiCall.Resource, "namespace", "n", "//./root/cimv2", "WMI namespace")
	wmiCallCmd.Flags().StringVarP(&wmiCall.Class, "class", "C", "", `WMI class to instantiate (i.e. "Win32_Process")`)
	wmiCallCmd.Flags().StringVarP(&wmiCall.Method, "method", "m", "", `WMI Method to call (i.e. "Create")`)
	wmiCallCmd.Flags().StringVarP(&wmiArguments, "args", "A", "{}", `WMI Method argument(s) in JSON dictionary format (i.e. {"CommandLine":"calc.exe"})`)

	if err := wmiCallCmd.MarkFlagRequired("class"); err != nil {
		panic(err)
	}
	if err := wmiCallCmd.MarkFlagRequired("method"); err != nil {
		panic(err)
	}
}

func wmiProcCmdInit() {
	wmiProcCmd.Flags().StringVarP(&wmiProc.Resource, "namespace", "n", "//./root/cimv2", "WMI namespace")
	wmiProcCmd.Flags().StringVarP(&wmiProc.WorkingDirectory, "directory", "d", `C:\`, "Working directory")

	registerProcessExecutionArgs(wmiProcCmd)
	registerExecutionOutputArgs(wmiProcCmd)
}

var (
	wmiCall = wmiexec.WmiCall{}
	wmiProc = wmiexec.WmiProc{}

	wmiArguments string

	wmiCmd = &cobra.Command{
		Use:   "wmi",
		Short: "Establish execution via wmi",
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
		Args: args(argsRpcClient("host"), wmiCallArgs),

		Run: func(cmd *cobra.Command, args []string) {
			var err error

			ctx := gssapi.NewSecurityContext(context.Background())

			ctx = log.With().
				Str("module", "wmi").
				Str("method", "call").
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

			if err = wmiCall.Init(ctx); err != nil {
				log.Error().Err(err).Msg("Module initialization failed")
				returnCode = 2
				return
			}

			out, err := wmiCall.Call(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Call failed")
				returnCode = 4
				return
			}
			fmt.Println(string(out))
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
			argsOutput("smb"),
			argsRpcClient("host"),
		),

		Run: func(cmd *cobra.Command, args []string) {
			wmiProc.Client = &rpcClient
			wmiProc.IO = exec

			ctx := log.With().
				Str("module", "wmi").
				Str("method", "proc").
				Logger().WithContext(gssapi.NewSecurityContext(context.TODO()))

			if err := goexec.ExecuteCleanMethod(ctx, &wmiProc, &exec); err != nil {
				log.Fatal().Err(err).Msg("Operation failed")
			}
		},
	}
)
