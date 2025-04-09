package cmd

import (
  "context"
  "fmt"
  "github.com/RedTeamPentesting/adauth"
  "github.com/rs/zerolog"
  "github.com/spf13/cobra"
  "net/url"
  "os"
  "regexp"
  "strings"
)

var (
  //logFile string
  log      zerolog.Logger
  ctx      context.Context
  authOpts *adauth.Options

  hostname string
  proxyStr string
  proxyUrl *url.URL

  // Root flags
  unsafe bool // not implemented
  debug  bool

  // Generic flags
  command          string
  executable       string
  executablePath   string
  executableArgs   string
  workingDirectory string
  windowState      string

  rootCmd = &cobra.Command{
    Use: "goexec",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
      // For modules that require a full executable path
      if executablePath != "" && !regexp.MustCompile(`^([a-zA-Z]:)?\\`).MatchString(executablePath) {
        return fmt.Errorf("executable path (-e) must be an absolute Windows path, i.e. C:\\Windows\\System32\\cmd.exe")
      }
      if command != "" {
        p := strings.SplitN(command, " ", 2)
        executable = p[0]
        if len(p) > 1 {
          executableArgs = p[1]
        }
      }
      log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
      if debug {
        log = log.Level(zerolog.DebugLevel)
      }
      return
    },
  }
)

func needs(reqs ...func(*cobra.Command, []string) error) (fn func(*cobra.Command, []string) error) {
  return func(cmd *cobra.Command, args []string) (err error) {
    for _, req := range reqs {
      if err = req(cmd, args); err != nil {
        return
      }
    }
    return
  }
}

func needsTarget(proto string) func(cmd *cobra.Command, args []string) error {

  return func(cmd *cobra.Command, args []string) (err error) {
    if proxyStr != "" {
      if proxyUrl, err = url.Parse(proxyStr); err != nil {
        return fmt.Errorf("failed to parse proxy URL %q: %w", proxyStr, err)
      }
    }
    if len(args) != 1 {
      return fmt.Errorf("command require exactly one positional argument: [target]")
    }
    if creds, target, err = authOpts.WithTarget(ctx, proto, args[0]); err != nil {
      return fmt.Errorf("failed to parse target: %w", err)
    }
    if creds == nil {
      return fmt.Errorf("no credentials supplied")
    }
    if target == nil {
      return fmt.Errorf("no target supplied")
    }
    if hostname, err = target.Hostname(ctx); err != nil {
      log.Debug().Err(err).Msg("Could not get target hostname")
    }
    return
  }
}

func init() {
  ctx = context.Background()

  cobra.EnableCommandSorting = false

  rootCmd.InitDefaultVersionFlag()
  rootCmd.InitDefaultHelpCmd()
  rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
  rootCmd.PersistentFlags().StringVarP(&proxyStr, "proxy", "x", "", "Proxy URL")
  rootCmd.PersistentFlags().BoolVar(&unsafe, "unsafe", false, "[NOT IMPLEMENTED] Don't ask for permission to run unsafe actions")

  authOpts = &adauth.Options{Debug: log.Debug().Msgf}
  authOpts.RegisterFlags(rootCmd.PersistentFlags())

  scmrCmdInit()
  rootCmd.AddCommand(scmrCmd)
  tschCmdInit()
  rootCmd.AddCommand(tschCmd)
  wmiCmdInit()
  rootCmd.AddCommand(wmiCmd)
  dcomCmdInit()
  rootCmd.AddCommand(dcomCmd)
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
