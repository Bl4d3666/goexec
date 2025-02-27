package cmd

import (
	"context"
	"fmt"
	"github.com/bryanmcnulty/adauth"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
	"regexp"
)

var (
	//logFile string
	log      zerolog.Logger
	ctx      context.Context
	authOpts *adauth.Options

	debug, trace   bool
	command        string
	executablePath string
	executableArgs string

	rootCmd = &cobra.Command{
		Use: "goexec",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			// For modules that require a full executable path
			if executablePath != "" && !regexp.MustCompile(`^([a-zA-Z]:)?\\`).MatchString(executablePath) {
				return fmt.Errorf("executable path (-e) must be an absolute Windows path, i.e. C:\\Windows\\System32\\cmd.exe")
			}
			log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
			if debug {
				log = log.Level(zerolog.DebugLevel)
			}
			return
		},
	}
)

func init() {
	ctx = context.Background()

	rootCmd.InitDefaultVersionFlag()
	rootCmd.InitDefaultHelpCmd()
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	authOpts = &adauth.Options{Debug: log.Debug().Msgf}
	authOpts.RegisterFlags(rootCmd.PersistentFlags())

	scmrCmdInit()
	rootCmd.AddCommand(scmrCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
