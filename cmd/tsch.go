package cmd

import (
	"fmt"
	"github.com/FalconOpsLLC/goexec/pkg/exec"
	tschexec "github.com/FalconOpsLLC/goexec/pkg/exec/tsch"
	"github.com/spf13/cobra"
	"time"
)

func tschCmdInit() {
	tschDeleteCmdInit()
	tschCmd.AddCommand(tschDeleteCmd)

	tschRegisterCmdInit()
	tschCmd.AddCommand(tschRegisterCmd)

	tschDemandCmdInit()
	tschCmd.AddCommand(tschDemandCmd)
}

func tschDeleteCmdInit() {
	tschDeleteCmd.Flags().StringVarP(&tschTaskPath, "path", "t", "", "Scheduled task path")
	tschDeleteCmd.MarkFlagRequired("path")
}

func tschDemandCmdInit() {
	tschDemandCmd.Flags().StringVarP(&executable, "executable", "e", "", "Remote Windows executable to invoke")
	tschDemandCmd.Flags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
	tschDemandCmd.Flags().StringVarP(&tschName, "name", "n", "", "Target task name")
	tschDemandCmd.Flags().BoolVar(&tschNoDelete, "no-delete", false, "Don't delete task after execution")
	tschDemandCmd.MarkFlagRequired("executable")
}

func tschRegisterCmdInit() {
	tschRegisterCmd.Flags().StringVarP(&executable, "executable", "e", "", "Remote Windows executable to invoke")
	tschRegisterCmd.Flags().StringVarP(&executableArgs, "args", "a", "", "Arguments to pass to executable")
	tschRegisterCmd.Flags().StringVarP(&tschName, "name", "n", "", "Target task name")
	tschRegisterCmd.Flags().DurationVar(&tschStopDelay, "delay-stop", time.Duration(5*time.Second), "Delay between task execution and termination. This will not stop the process spawned by the task")
	tschRegisterCmd.Flags().DurationVarP(&tschDelay, "delay-start", "d", time.Duration(5*time.Second), "Delay between task registration and execution")
	tschRegisterCmd.Flags().DurationVarP(&tschDeleteDelay, "delay-delete", "D", time.Duration(0*time.Second), "Delay between task termination and deletion")
	tschRegisterCmd.Flags().BoolVar(&tschNoDelete, "no-delete", false, "Don't delete task after execution")
	tschRegisterCmd.Flags().BoolVar(&tschCallDelete, "call-delete", false, "Directly call SchRpcDelete to delete task")

	tschRegisterCmd.MarkFlagsMutuallyExclusive("no-delete", "delay-delete")
	tschRegisterCmd.MarkFlagsMutuallyExclusive("no-delete", "call-delete")
	tschRegisterCmd.MarkFlagsMutuallyExclusive("delay-delete", "call-delete")
	tschRegisterCmd.MarkFlagRequired("executable")
}

var (
	tschNoDelete    bool
	tschCallDelete  bool
	tschDeleteDelay time.Duration
	tschStopDelay   time.Duration
	tschDelay       time.Duration
	tschName        string
	tschTaskPath    string

	tschCmd = &cobra.Command{
		Use:   "tsch",
		Short: "Establish execution via TSCH (ITaskSchedulerService)",
		Args: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("command not set. Choose from (delete, register, demand)")
		},
	}
	tschRegisterCmd = &cobra.Command{
		Use:   "register [target]",
		Short: "Register a scheduled task with an automatic start time",
		Args:  needsTarget,
		Run: func(cmd *cobra.Command, args []string) {
			if tschNoDelete {
				log.Warn().Msg("Task will not be deleted after execution")
			}
			module := tschexec.Module{}
			execCfg := &exec.ExecutionConfig{
				ExecutableName:  executable,
				ExecutableArgs:  executableArgs,
				ExecutionMethod: tschexec.MethodRegister,

				ExecutionMethodConfig: tschexec.MethodRegisterConfig{
					NoDelete:    tschNoDelete,
					CallDelete:  tschCallDelete,
					StartDelay:  tschDelay,
					StopDelay:   tschStopDelay,
					DeleteDelay: tschDeleteDelay,
					TaskName:    tschName,
				},
			}
			if err := module.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
				log.Fatal().Err(err).Msg("TSCH execution failed")
			}
		},
	}
	tschDemandCmd = &cobra.Command{
		Use:   "demand [target]",
		Short: "Register a scheduled task and demand immediate start",
		Args:  needsTarget,
		Run: func(cmd *cobra.Command, args []string) {
			if tschNoDelete {
				log.Warn().Msg("Task will not be deleted after execution")
			}
			module := tschexec.Module{}
			execCfg := &exec.ExecutionConfig{
				ExecutableName:  executable,
				ExecutableArgs:  executableArgs,
				ExecutionMethod: tschexec.MethodDemand,

				ExecutionMethodConfig: tschexec.MethodDemandConfig{
					NoDelete: tschNoDelete,
					TaskName: tschName,
				},
			}
			if err := module.Exec(log.WithContext(ctx), creds, target, execCfg); err != nil {
				log.Fatal().Err(err).Msg("TSCH execution failed")
			}
		},
	}
	tschDeleteCmd = &cobra.Command{
		Use:   "delete [target]",
		Short: "Delete a scheduled task",
		Args:  needsTarget,
		Run: func(cmd *cobra.Command, args []string) {
			module := tschexec.Module{}
			cleanCfg := &exec.CleanupConfig{
				CleanupMethod:       tschexec.MethodDelete,
				CleanupMethodConfig: tschexec.MethodDeleteConfig{TaskPath: tschTaskPath},
			}
			if err := module.Cleanup(log.WithContext(ctx), creds, target, cleanCfg); err != nil {
				log.Fatal().Err(err).Msg("TSCH cleanup failed")
			}
		},
	}
)
