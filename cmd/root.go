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
  "os"
)

var (
  debug        bool
  logJson      bool
  returnCode   int
  outputMethod string
  outputPath   string
  proxy        string

  log zerolog.Logger

  rpcClient dce.Client
  smbClient smb.Client

  exec = goexec.ExecutionIO{
    Input:  new(goexec.ExecutionInput),
    Output: new(goexec.ExecutionOutput),
  }

  authOpts   *adauth.Options
  credential *adauth.Credential
  target     *adauth.Target

  rootCmd = &cobra.Command{
    Use:   "goexec",
    Short: `Windows remote execution multitool`,
    Long:  `TODO`,

    PersistentPreRun: func(cmd *cobra.Command, args []string) {

      if logJson {
        log = zerolog.New(os.Stderr)
      } else {
        log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
      }

      log = log.Level(zerolog.InfoLevel).With().Timestamp().Logger()
      if debug {
        log = log.Level(zerolog.DebugLevel)
      }

      if outputMethod == "smb" {
        if exec.Output.RemotePath == "" {
          exec.Output.RemotePath = util.RandomWindowsTempFile()
        }
        exec.Output.Provider = &smb.OutputFileFetcher{
          Client: &smbClient,
          Share:  `C$`,
          File:   exec.Output.RemotePath,
        }
      }
    },
  }
)

func init() {
  // Cobra init
  {
    cobra.EnableCommandSorting = false

    rootCmd.InitDefaultVersionFlag()
    rootCmd.InitDefaultHelpCmd()
    rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
    rootCmd.PersistentFlags().BoolVar(&logJson, "log-json", false, "Log in JSON format")

    dcomCmdInit()
    rootCmd.AddCommand(dcomCmd)

    wmiCmdInit()
    rootCmd.AddCommand(wmiCmd)

    scmrCmdInit()
    rootCmd.AddCommand(scmrCmd)

    tschCmdInit()
    rootCmd.AddCommand(tschCmd)
  }

  // Auth init
  {
    gssapi.AddMechanism(ssp.SPNEGO)
    gssapi.AddMechanism(ssp.NTLM)
    gssapi.AddMechanism(ssp.KRB5)

    authOpts = &adauth.Options{Debug: log.Debug().Msgf}
    authOpts.RegisterFlags(rootCmd.PersistentFlags())
  }
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  os.Exit(returnCode)
}
