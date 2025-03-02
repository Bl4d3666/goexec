package cmd

import (
	"github.com/bryanmcnulty/adauth"
	"github.com/spf13/cobra"

	"github.com/FalconOpsLLC/goexec/pkg/exec"
	scmrexec "github.com/FalconOpsLLC/goexec/pkg/exec/scmr"
	"github.com/FalconOpsLLC/goexec/pkg/windows"
)

func scmrCmdInit() {
	scmrCmd.PersistentFlags().StringVarP(&executablePath, "executable-path", "f", "", "Full path to remote Windows executable")
	scmrCmd.PersistentFlags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
	scmrCmd.PersistentFlags().StringVarP(&scmrName, "service-name", "s", "", "Name of service to create or modify")

	scmrCmd.MarkPersistentFlagRequired("executable-path")
	scmrCmd.MarkPersistentFlagRequired("service-name")

	scmrChangeCmdInit()
	scmrCmd.AddCommand(scmrChangeCmd)

	scmrCreateCmdInit()
	scmrCmd.AddCommand(scmrCreateCmd)
}

func scmrChangeCmdInit() {
	scmrChangeCmd.Flags().StringVarP(&scmrDisplayName, "display-name", "n", "", "Display name of service to create")
	scmrChangeCmd.Flags().BoolVar(&scmrNoStart, "no-start", false, "Don't start service")
}

func scmrCreateCmdInit() {
	scmrCreateCmd.Flags().BoolVar(&scmrNoDelete, "no-delete", false, "Don't delete service after execution")
}

var (
	// scmr arguments
	scmrName        string
	scmrDisplayName string
	scmrNoDelete    bool
	scmrNoStart     bool

	creds  *adauth.Credential
	target *adauth.Target

	scmrCmd = &cobra.Command{
		Use:   "scmr",
		Short: "Establish execution via SCMR",
	}
	scmrCreateCmd = &cobra.Command{
		Use:   "create [target]",
		Short: "Create & run a new Windows service to gain execution",
		Args:  needsTarget,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if scmrNoDelete {
				log.Warn().Msg("Service will not be deleted after execution")
			}
			if scmrDisplayName == "" {
				scmrDisplayName = scmrName
				log.Warn().Msg("No display name specified, using service name as display name")
			}
			module := scmrexec.Module{}
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
			if err := module.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
				log.Fatal().Err(err).Msg("SCMR execution failed")
			}
			return nil
		},
	}
	scmrChangeCmd = &cobra.Command{
		Use:   "change [target]",
		Short: "Change an existing Windows service to gain execution",
		Args:  needsTarget,
		Run: func(cmd *cobra.Command, args []string) {
			executor := scmrexec.Module{}
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
