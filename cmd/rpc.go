package cmd

import (
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/spf13/cobra"
  "regexp"
)

func needsRpcTarget(proto string) func(cmd *cobra.Command, args []string) error {
  return func(cmd *cobra.Command, args []string) (err error) {

    if argDceStringBinding != "" {
      dceConfig.Endpoint, err = dcerpc.ParseStringBinding(argDceStringBinding)
      if err != nil {
        return fmt.Errorf("failed to parse RPC endpoint: %w", err)
      }
      dceConfig.NoEpm = true // If an explicit endpoint is set, don't use EPM

    } else if argDceEpmFilter != "" {
      // This ensures that filters like "ncacn_ip_tcp" will be made into a valid binding (i.e. "ncacn_ip_tcp:")
      if ok, err := regexp.MatchString(`^\w+$`, argDceEpmFilter); err == nil && ok {
        argDceEpmFilter += ":"
      }
      dceConfig.Endpoint, err = dcerpc.ParseStringBinding(argDceEpmFilter)
      if err != nil {
        return fmt.Errorf("failed to parse EPM filter: %w", err)
      }
    }
    if !argDceNoSign {
      dceConfig.DceOptions = append(dceConfig.DceOptions, dcerpc.WithSign())
      dceConfig.EpmOptions = append(dceConfig.EpmOptions, dcerpc.WithSign())
    }
    if argDceNoSeal {
      dceConfig.DceOptions = append(dceConfig.DceOptions, dcerpc.WithInsecure())
    } else {
      dceConfig.DceOptions = append(dceConfig.DceOptions, dcerpc.WithSeal(), dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy))
      dceConfig.EpmOptions = append(dceConfig.EpmOptions, dcerpc.WithSeal(), dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy))
    }
    return needsTarget(proto)(cmd, args)
  }
}

var (
  // DCE arguments
  argDceStringBinding string
  argDceEpmFilter     string
  argDceNoSeal        bool
  argDceNoSign        bool

  // DCE options
  dceStringBinding *dcerpc.StringBinding
  dceConfig        dce.ConnectionMethodDCEConfig
)

func registerRpcFlags(cmd *cobra.Command) {
  cmd.PersistentFlags().BoolVar(&dceConfig.NoEpm, "no-epm", false, "Do not use EPM to automatically detect endpoints")
  cmd.PersistentFlags().BoolVar(&dceConfig.EpmAuto, "epm-auto", false, "Automatically detect endpoints instead of using the module defaults")
  cmd.PersistentFlags().BoolVar(&argDceNoSign, "no-sign", false, "Disable signing on DCE messages")
  cmd.PersistentFlags().BoolVar(&argDceNoSeal, "no-seal", false, "Disable packet stub encryption on DCE messages")
  cmd.PersistentFlags().StringVarP(&argDceEpmFilter, "epm-filter", "F", "", "String binding to filter endpoints returned by EPM")
  cmd.PersistentFlags().StringVar(&argDceStringBinding, "endpoint", "", "Explicit RPC endpoint definition")

  cmd.MarkFlagsMutuallyExclusive("endpoint", "epm-filter")
  cmd.MarkFlagsMutuallyExclusive("no-epm", "epm-filter")
}
