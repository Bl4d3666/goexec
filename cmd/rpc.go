package cmd

import (
  "fmt"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/spf13/cobra"
  "regexp"
)

var (
  // DCE options
  argDceStringBinding string
  argDceEpmFilter     string
  argDceEpmAuto       bool
  argDceNoEpm         bool
  argDceNoSeal        bool
  argDceNoSign        bool
  dceStringBinding    *dcerpc.StringBinding
  dceOptions          []dcerpc.Option

  needsRpcTarget = func(proto string) func(cmd *cobra.Command, args []string) error {
    return func(cmd *cobra.Command, args []string) (err error) {
      if argDceStringBinding != "" {
        dceStringBinding, err = dcerpc.ParseStringBinding(argDceStringBinding)
        if err != nil {
          return fmt.Errorf("failed to parse RPC endpoint: %w", err)
        }
        argDceNoEpm = true // If an explicit endpoint is set, don't use EPM

      } else if argDceEpmFilter != "" {
        // This ensures that filters like "ncacn_ip_tcp" will be made into a valid binding (i.e. "ncacn_ip_tcp:")
        if ok, err := regexp.MatchString(`^\w+$`, argDceEpmFilter); err == nil && ok {
          argDceEpmFilter += ":"
        }
        dceStringBinding, err = dcerpc.ParseStringBinding(argDceEpmFilter)
        if err != nil {
          return fmt.Errorf("failed to parse EPM filter: %w", err)
        }
      }
      if !argDceNoSign {
        dceOptions = append(dceOptions, dcerpc.WithSign())
      }
      if argDceNoSeal {
        dceOptions = append(dceOptions, dcerpc.WithInsecure())
      } else {
        dceOptions = append(dceOptions, dcerpc.WithSeal())
      }
      return needsTarget(proto)(cmd, args)
    }
  }
)

func registerRpcFlags(cmd *cobra.Command) {
  cmd.PersistentFlags().StringVarP(&argDceEpmFilter, "epm-filter", "F", "", "String binding to filter endpoints returned by EPM")
  cmd.PersistentFlags().StringVar(&argDceStringBinding, "endpoint", "", "Explicit RPC endpoint definition")
  cmd.PersistentFlags().BoolVar(&argDceNoEpm, "no-epm", false, "Do not use EPM to automatically detect endpoints")
  cmd.PersistentFlags().BoolVar(&argDceNoSign, "no-sign", false, "Disable signing on DCE messages")
  cmd.PersistentFlags().BoolVar(&argDceNoSeal, "no-seal", false, "Disable packet stub encryption on DCE messages")
  cmd.PersistentFlags().BoolVar(&argDceEpmAuto, "epm-auto", false, "Automatically detect endpoints instead of using the module defaults")
  cmd.MarkFlagsMutuallyExclusive("endpoint", "epm-filter")
  cmd.MarkFlagsMutuallyExclusive("no-epm", "epm-filter")
}
