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

type flagSet struct {
	Label string
	Flags *pflag.FlagSet
}

const helpTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command] [flags]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
{{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if (ne .Name "completion")}}{{range $_, $v := cmdFlags .}}

{{$v.Label|trimTrailingWhitespaces}}:
{{flags $v.Flags|trimTrailingWhitespaces}}{{end}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var (
	cmdFlags = make(map[*cobra.Command][]*flagSet)

	defaultAuthFlags, defaultLogFlags, defaultNetRpcFlags *flagSet

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
		Short: `goexec - Windows remote execution multitool`,
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

func newFlagSet(name string) *flagSet {
	flags := pflag.NewFlagSet(name, pflag.ExitOnError)
	flags.SortFlags = false
	return &flagSet{
		Label: name,
		Flags: flags,
	}
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
		{
			defaultNetRpcFlags = newFlagSet("Network")
			registerNetworkFlags(defaultNetRpcFlags.Flags)
		}
		{
			defaultLogFlags = newFlagSet("Logging")
			registerLoggingFlags(defaultLogFlags.Flags)
		}
		{
			defaultAuthFlags = newFlagSet("Authentication")
			adAuthOpts = &adauth.Options{
				Debug: log.Debug().Msgf,
			}
			adAuthOpts.RegisterFlags(defaultAuthFlags.Flags)
		}

		modules := &cobra.Group{
			ID:    "module",
			Title: "Execution Commands:",
		}
		rootCmd.AddGroup(modules)

		cmdFlags[rootCmd] = []*flagSet{
			defaultLogFlags,
			defaultAuthFlags,
		}

		cobra.AddTemplateFunc("flags", func(fs *pflag.FlagSet) string {
			if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				return fs.FlagUsagesWrapped(width - 1)
			}
			return fs.FlagUsagesWrapped(80 - 1)
		})

		cobra.AddTemplateFunc("cmdFlags", func(cmd *cobra.Command) []*flagSet {
			return cmdFlags[cmd]
		})

		rootCmd.InitDefaultVersionFlag()
		rootCmd.InitDefaultHelpCmd()
		rootCmd.SetHelpTemplate("{{if (ne .Long \"\")}}{{.Long}}\n{{end}}" + helpTemplate)
		rootCmd.SetUsageTemplate(helpTemplate)

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
