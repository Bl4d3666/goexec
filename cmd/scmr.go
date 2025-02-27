package cmd

import (
	"errors"
	"fmt"
	"github.com/bryanmcnulty/adauth"
	"github.com/spf13/cobra"

	"github.com/FalconOpsLLC/goexec/pkg/exec"
	scmrexec "github.com/FalconOpsLLC/goexec/pkg/exec/scmr"
	"github.com/FalconOpsLLC/goexec/pkg/windows"
)

func scmrCmdInit() {
	scmrCmd.PersistentFlags().StringVarP(&executablePath, "executable-path", "e", "", "Full path to remote Windows executable")
	scmrCmd.PersistentFlags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
	scmrCmd.PersistentFlags().StringVarP(&scmrName, "service", "s", "", "Name of service to create or modify")
	scmrCmd.PersistentFlags().StringVarP(&scmrDisplayName, "display-name", "n", "", "Service display name")

	scmrCmd.MarkPersistentFlagRequired("executable-path")
	scmrCmd.MarkPersistentFlagRequired("service")

	scmrCmd.AddCommand(scmrChangeCmd)
	scmrChangeCmdInit()
	scmrCmd.AddCommand(scmrCreateCmd)
	scmrCreateCmdInit()
}

func scmrChangeCmdInit() {
	scmrChangeCmd.Flags().BoolVar(&scmrNoStart, "no-start", false, "Don't start service")
}

func scmrCreateCmdInit() {
	scmrChangeCmd.Flags().StringVarP(&scmrDisplayName, "display-name", "n", "", "Display name of service to create")
	scmrCreateCmd.Flags().BoolVar(&scmrNoDelete, "no-delete", false, "Don't delete service after execution")
}

var (
	// scmr arguments
	scmrName        string
	scmrDisplayName string
	scmrNoDelete    bool
	scmrNoStart     bool

	scmrArgs = func(cmd *cobra.Command, args []string) (err error) {
		if len(args) != 1 {
			return fmt.Errorf("expected exactly 1 positional argument, got %d", len(args))
		}
		if creds, target, err = authOpts.WithTarget(ctx, "cifs", args[0]); err != nil {
			return fmt.Errorf("failed to parse target: %w", err)
		}
		log.Debug().Str("target", args[0]).Msg("Resolved target")
		return nil
	}

	creds  *adauth.Credential
	target *adauth.Target

	scmrCmd = &cobra.Command{
		Use:   "scmr",
		Short: "Establish execution via SCMR",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New(`command not set. Choose from (change, create)`)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				panic(err)
			}
		},
	}
	scmrCreateCmd = &cobra.Command{
		Use:   "create [target]",
		Short: "Create & run a new Windows service to gain execution",
		Args:  scmrArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if scmrNoDelete {
				log.Warn().Msg("Service will not be deleted after execution")
			}
			if scmrDisplayName == "" {
				scmrDisplayName = scmrName
				log.Warn().Msg("No display name specified, using service name as display name")
			}
			executor := scmrexec.Executor{}
			execCfg := &exec.ExecutionConfig{
				ExecutablePath:  executablePath,
				ExecutableArgs:  executableArgs,
				ExecutionMethod: scmrexec.MethodCreate,

				ExecutionMethodConfig: scmrexec.MethodCreateConfig{
					NoDelete:    scmrNoDelete,
					ServiceName: scmrName,
					DisplayName: scmrDisplayName,
					ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
					StartType:   windows.SERVICE_DEMAND_START,
				},
			}
			if err := executor.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
				log.Fatal().Err(err).Msg("SCMR execution failed")
			}
			return nil
		},
	}
	scmrChangeCmd = &cobra.Command{
		Use:   "change [target]",
		Short: "Change an existing Windows service to gain execution",
		Args:  scmrArgs,
		Run: func(cmd *cobra.Command, args []string) {
			executor := scmrexec.Executor{}
			execCfg := &exec.ExecutionConfig{
				ExecutablePath:  executablePath,
				ExecutableArgs:  executableArgs,
				ExecutionMethod: scmrexec.MethodModify,

				ExecutionMethodConfig: scmrexec.MethodModifyConfig{
					NoStart:     scmrNoStart,
					ServiceName: scmrName,
				},
			}
			if err := executor.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
				log.Fatal().Err(err).Msg("SCMR execution failed")
			}
		},
	}
)
