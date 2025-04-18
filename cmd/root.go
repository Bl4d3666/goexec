package cmd

import (
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/FalconOpsLLC/goexec/pkg/goexec/dce"
  "github.com/FalconOpsLLC/goexec/pkg/goexec/smb"
  "github.com/RedTeamPentesting/adauth"
  "github.com/oiweiwei/go-msrpc/ssp"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/rs/zerolog"
  "github.com/spf13/cobra"
  "github.com/spf13/pflag"
  "golang.org/x/term"
  "io"
  "os"
)

var (
  flagGroups = map[string]*pflag.FlagSet{}

  returnCode   int
  outputMethod string
  outputPath   string
  proxy        string

  // === Logging ===
  logJson   bool           // Log output in JSON lines
  logDebug  bool           // Output debug log messages
  logQuiet  bool           // Suppress logging output
  logOutput string         // Log output file
  logLevel  zerolog.Level  = zerolog.InfoLevel
  logFile   io.WriteCloser = os.Stderr
  log       zerolog.Logger
  // ===============

  rpcClient dce.Client
  smbClient smb.Client

  exec = goexec.ExecutionIO{
    Input:  new(goexec.ExecutionInput),
    Output: new(goexec.ExecutionOutput),
  }

  adAuthOpts *adauth.Options
  credential *adauth.Credential
  target     *adauth.Target

  rootCmd = &cobra.Command{
    Use:   "goexec",
    Short: `Windows remote execution multitool`,
    Long:  ``,

    PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {

      // Parse logging options
      {
        if logOutput != "" {
          logFile, err = os.OpenFile(logOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
          if err != nil {
            return
          }
          logJson = true
        }
        if logQuiet {
          logLevel = zerolog.ErrorLevel
        } else if logDebug {
          logLevel = zerolog.DebugLevel
        }
        if logJson {
          log = zerolog.New(logFile).With().Timestamp().Logger()
        } else {
          log = zerolog.New(zerolog.ConsoleWriter{Out: logFile}).With().Timestamp().Logger()
        }
        log = log.Level(logLevel)
      }

      if proxy != "" {
        rpcClient.Proxy = proxy
        smbClient.Proxy = proxy
      }

      if outputPath != "" {
        if outputMethod == "smb" {
          if exec.Output.RemotePath == "" {
            exec.Output.RemotePath = util.RandomWindowsTempFile()
          }
          exec.Output.Provider = &smb.OutputFileFetcher{
            Client:           &smbClient,
            Share:            `C$`,
            File:             exec.Output.RemotePath,
            DeleteOutputFile: !exec.Output.NoDelete,
          }
        }
      }
      return
    },

    PersistentPostRun: func(cmd *cobra.Command, args []string) {
      if err := logFile.Close(); err != nil {
        // ...
      }
    },
  }
)

func addFlagSet(fs *pflag.FlagSet) {
  flagGroups[fs.Name()] = fs
}

func moduleFlags(cmd *cobra.Command, module string) (fs *pflag.FlagSet) {
  fs, _ = flagGroups[module]
  return
}

// Uses the users terminal size or width of 80 if cannot determine users width
// Based on https://github.com/spf13/cobra/issues/1805#issuecomment-1246192724
func wrappedFlagUsages(cmd *pflag.FlagSet) string {
  fd := int(os.Stdout.Fd())
  width := 80

  // Get the terminal width and dynamically set
  termWidth, _, err := term.GetSize(fd)
  if err == nil {
    width = termWidth
  }

  return cmd.FlagUsagesWrapped(width - 1)
}

func init() {
  // Auth init
  {
    gssapi.AddMechanism(ssp.SPNEGO)
    gssapi.AddMechanism(ssp.NTLM)
    gssapi.AddMechanism(ssp.KRB5)
  }

  // Cobra init
  {
    cobra.EnableCommandSorting = false

    rootCmd.InitDefaultVersionFlag()
    rootCmd.InitDefaultHelpCmd()

    modules := &cobra.Group{
      ID:    "module",
      Title: "Execution Commands:",
    }
    rootCmd.AddGroup(modules)

    // Logging flags
    {
      logOpts := pflag.NewFlagSet("Logging", pflag.ExitOnError)
      logOpts.BoolVar(&logDebug, "debug", false, "Enable debug logging")
      logOpts.BoolVar(&logJson, "json", false, "Write logging output in JSON lines")
      logOpts.BoolVar(&logQuiet, "quiet", false, "Disable info logging")
      logOpts.StringVarP(&logOutput, "log-file", "O", "", "Write JSON logging output to `file`")
      rootCmd.PersistentFlags().AddFlagSet(logOpts)
      flagGroups["Logging"] = logOpts
    }

    // Global networking flags
    {
      netOpts := pflag.NewFlagSet("Network", pflag.ExitOnError)
      netOpts.StringVarP(&proxy, "proxy", "x", "", "Proxy `URI`")
      rootCmd.PersistentFlags().AddFlagSet(netOpts)
    }

    // Authentication flags
    {
      adAuthOpts = &adauth.Options{
        Debug: log.Debug().Msgf,
      }
      authOpts := pflag.NewFlagSet("Authentication", pflag.ExitOnError)
      adAuthOpts.RegisterFlags(authOpts)
      rootCmd.PersistentFlags().AddFlagSet(authOpts)
    }

    // Modules init
    {
      dcomCmdInit()
      rootCmd.AddCommand(dcomCmd)
      wmiCmdInit()
      rootCmd.AddCommand(wmiCmd)
      scmrCmdInit()
      rootCmd.AddCommand(scmrCmd)
      tschCmdInit()
      rootCmd.AddCommand(tschCmd)
    }
  }
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  os.Exit(returnCode)
}
