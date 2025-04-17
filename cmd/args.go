package cmd

import (
  "context"
  "errors"
  "fmt"
  "github.com/spf13/cobra"
  "github.com/spf13/pflag"
  "os"
)

func registerRpcFlags(cmd *cobra.Command) {
  rpcFlags := pflag.NewFlagSet("RPC", pflag.ExitOnError)

  rpcFlags.BoolVar(&rpcClient.NoEpm, "no-epm", false, "Do not use EPM to automatically detect endpoints")
  //rpcFlags.BoolVar(&rpcClient.Options.EpmAuto, "epm-auto", false, "Automatically detect endpoints instead of using the module defaults")
  rpcFlags.BoolVar(&rpcClient.NoSign, "no-sign", false, "Disable signing on DCE messages")
  rpcFlags.BoolVar(&rpcClient.NoSeal, "no-seal", false, "Disable packet stub encryption on DCE messages")
  rpcFlags.StringVar(&rpcClient.Filter, "epm-filter", "", "String binding to filter endpoints returned by EPM")
  rpcFlags.StringVar(&rpcClient.Endpoint, "endpoint", "", "Explicit RPC endpoint definition")

  cmd.PersistentFlags().AddFlagSet(rpcFlags)

  cmd.MarkFlagsMutuallyExclusive("endpoint", "epm-filter")
  cmd.MarkFlagsMutuallyExclusive("no-epm", "epm-filter")
}

func registerProcessExecutionArgs(cmd *cobra.Command) {
  group := pflag.NewFlagSet("Execution", pflag.ExitOnError)

  group.StringVarP(&exec.Input.Arguments, "args", "a", "", "Command line arguments")
  group.StringVarP(&exec.Input.CommandLine, "command", "c", "", "Windows process command line (executable & arguments)")
  group.StringVarP(&exec.Input.Executable, "executable", "e", "", "Windows executable to invoke")

  cmd.PersistentFlags().AddFlagSet(group)

  cmd.MarkFlagsOneRequired("executable", "command")
  cmd.MarkFlagsMutuallyExclusive("executable", "command")
}

func registerExecutionOutputArgs(cmd *cobra.Command) {
  group := pflag.NewFlagSet("Output", pflag.ExitOnError)

  group.StringVarP(&outputPath, "output", "o", "", `Fetch execution output to file or "-" for standard output`)
  group.StringVarP(&outputMethod, "output-method", "m", "smb", "Method to fetch execution output")
  group.StringVar(&exec.Output.RemotePath, "remote-output", "", "Location to temporarily store output on remote filesystem")
  group.BoolVar(&exec.Output.NoDelete, "no-delete-output", false, "Preserve output file on remote filesystem")

  cmd.PersistentFlags().AddFlagSet(group)
}

func args(reqs ...func(*cobra.Command, []string) error) (fn func(*cobra.Command, []string) error) {

  return func(cmd *cobra.Command, args []string) (err error) {

    for _, req := range reqs {
      if err = req(cmd, args); err != nil {
        return
      }
    }
    return
  }
}

func argsTarget(proto string) func(cmd *cobra.Command, args []string) error {

  return func(cmd *cobra.Command, args []string) (err error) {

    if len(args) != 1 {
      return errors.New("command require exactly one positional argument: [target]")
    }

    if credential, target, err = authOpts.WithTarget(context.TODO(), proto, args[0]); err != nil {
      return fmt.Errorf("failed to parse target: %w", err)
    }

    if credential == nil {
      return errors.New("no credentials supplied")
    }
    if target == nil {
      return errors.New("no target supplied")
    }
    return
  }
}

func argsSmbClient() func(cmd *cobra.Command, args []string) error {
  return args(
    argsTarget("cifs"),

    func(_ *cobra.Command, _ []string) error {

      smbClient.Credential = credential
      smbClient.Target = target
      smbClient.Proxy = proxy

      return smbClient.Parse(context.TODO())
    },
  )
}

func argsRpcClient(proto string) func(cmd *cobra.Command, args []string) error {
  return args(
    argsTarget(proto),

    func(cmd *cobra.Command, args []string) (err error) {

      rpcClient.Target = target
      rpcClient.Credential = credential
      rpcClient.Proxy = proxy

      return rpcClient.Parse(context.TODO())
    },
  )
}

func argsOutput(methods ...string) func(cmd *cobra.Command, args []string) error {

  var as []func(*cobra.Command, []string) error

  for _, method := range methods {
    if method == "smb" {
      as = append(as, argsSmbClient())
    }
  }

  return args(append(as, func(*cobra.Command, []string) (err error) {

    if outputPath != "" {
      if outputPath == "-" {
        exec.Output.Writer = os.Stdout

      } else if outputPath != "" {

        if exec.Output.Writer, err = os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE, 0644); err != nil {
          log.Fatal().Err(err).Msg("Failed to open output file")
        }
      }
    }
    return
  })...)
}
