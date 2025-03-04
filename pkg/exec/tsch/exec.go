package tschexec

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/FalconOpsLLC/goexec/internal/util"
	dce "github.com/FalconOpsLLC/goexec/pkg/client/dcerpc"
	"github.com/FalconOpsLLC/goexec/pkg/exec"
	"github.com/RedTeamPentesting/adauth"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
	"github.com/rs/zerolog"
	"regexp"
	"time"
)

const (
	TaskXMLDurationFormat = "2006-01-02T15:04:05.9999999Z"
	TaskXMLHeader         = `<?xml version="1.0" encoding="UTF-16"?>`
)

var (
	TaskPathRegex = regexp.MustCompile(`^\\[^ :/\\][^:/]*$`) // https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/fa8809c8-4f0f-4c6d-994a-6c10308757c1
	TaskNameRegex = regexp.MustCompile(`^[^ :/\\][^:/\\]*$`)
)

// *very* simple implementation of xs:duration - only accepts +seconds
func xmlDuration(dur time.Duration) string {
	if s := int(dur.Seconds()); s >= 0 {
		return fmt.Sprintf(`PT%dS`, s)
	}
	return `PT0S`
}

// Connect to the target & initialize DCE & TSCH clients
func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target) (err error) {
	if mod.dce == nil {
		mod.dce = dce.NewDCEClient(ctx, false, &dce.SmbConfig{})
		if err = mod.dce.Connect(ctx, creds, target); err != nil {
			return fmt.Errorf("DCE connect: %w", err)
		} else if mod.tsch, err = itaskschedulerservice.NewTaskSchedulerServiceClient(ctx, mod.dce.DCE(), dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy)); err != nil {
			return fmt.Errorf("init MS-TSCH client: %w", err)
		}
		mod.log.Info().Msg("DCE connection successful")
	}
	return
}

func (mod *Module) Cleanup(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.CleanupConfig) (err error) {
	mod.log = zerolog.Ctx(ctx).With().
		Str("module", "tsch").
		Str("method", ccfg.CleanupMethod).Logger()
	mod.creds = creds
	mod.target = target

	if ccfg.CleanupMethod == MethodDelete {
		if cfg, ok := ccfg.CleanupMethodConfig.(MethodDeleteConfig); !ok {
			return errors.New("invalid configuration")
		} else {
			if err = mod.Connect(ctx, creds, target); err != nil {
				return fmt.Errorf("connect: %w", err)
			} else if _, err = mod.tsch.Delete(ctx, &itaskschedulerservice.DeleteRequest{
				Path:  cfg.TaskPath,
				Flags: 0,
			}); err != nil {
				mod.log.Error().Err(err).Str("task", cfg.TaskPath).Msg("Failed to delete task")
				return fmt.Errorf("delete task: %w", err)
			} else {
				mod.log.Info().Str("task", cfg.TaskPath).Msg("Task deleted successfully")
			}
		}
	} else {
		return fmt.Errorf("method not implemented: %s", ccfg.CleanupMethod)
	}
	return
}

func (mod *Module) Exec(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ecfg *exec.ExecutionConfig) (err error) {

	mod.log = zerolog.Ctx(ctx).With().
		Str("module", "tsch").
		Str("method", ecfg.ExecutionMethod).Logger()
	mod.creds = creds
	mod.target = target

	if ecfg.ExecutionMethod == MethodRegister {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodRegisterConfig); !ok {
			return errors.New("invalid configuration")

		} else {
			startTime := time.Now().UTC().Add(cfg.StartDelay)
			task := &task{
				TaskVersion:   "1.2",                                                   // static
				TaskNamespace: "http://schemas.microsoft.com/windows/2004/02/mit/task", // static
				TimeTriggers: []taskTimeTrigger{
					{
						StartBoundary: startTime.Format(TaskXMLDurationFormat),
						Enabled:       true,
					},
				},
				Principals: defaultPrincipals,
				Settings:   defaultSettings,
				Actions:    actions{Context: defaultPrincipals.Principals[0].ID, Exec: []actionExec{{Command: ecfg.ExecutableName, Arguments: ecfg.ExecutableArgs}}},
			}
			if !cfg.NoDelete && !cfg.CallDelete {
				if cfg.StopDelay == 0 {
					// EndBoundary cannot be >= StartBoundary
					cfg.StopDelay = 1 * time.Second
				}
				stopTime := startTime.Add(cfg.StopDelay)

				mod.log.Info().Time("when", stopTime).Msg("Task is scheduled to delete")
				task.Settings.DeleteExpiredTaskAfter = xmlDuration(cfg.DeleteDelay)
				task.TimeTriggers[0].EndBoundary = stopTime.Format(TaskXMLDurationFormat)
			}

			if doc, err := xml.Marshal(task); err != nil {
				return fmt.Errorf("marshal task XML: %w", err)

			} else {
				mod.log.Debug().Str("task", string(doc)).Msg("Task XML generated")
				docStr := TaskXMLHeader + string(doc)

				taskPath := cfg.TaskPath
				taskName := cfg.TaskName

				if taskName == "" {
					taskName = util.RandomString()
				}
				if taskPath == "" {
					taskPath = `\` + taskName
				}

				if err = mod.Connect(ctx, creds, target); err != nil {
					return fmt.Errorf("connect: %w", err)
				}
				defer func() {
					if err = mod.dce.Close(ctx); err != nil {
						mod.log.Warn().Err(err).Msg("Failed to dispose dce client")
					} else {
						mod.log.Debug().Msg("Disposed DCE client")
					}
				}()
				var response *itaskschedulerservice.RegisterTaskResponse
				if response, err = mod.tsch.RegisterTask(ctx, &itaskschedulerservice.RegisterTaskRequest{
					Path:       taskPath,
					XML:        docStr,
					Flags:      0, // TODO
					LogonType:  0, // TASK_LOGON_NONE
					CredsCount: 0,
					Creds:      nil,
				}); err != nil {
					return err

				} else {
					mod.log.Info().Str("path", response.ActualPath).Msg("Task registered successfully")

					if !cfg.NoDelete && cfg.CallDelete {
						defer func() {
							if err = mod.Cleanup(ctx, creds, target, &exec.CleanupConfig{
								CleanupMethod:       MethodDelete,
								CleanupMethodConfig: MethodDeleteConfig{TaskPath: taskPath},
							}); err != nil {
								mod.log.Error().Err(err).Msg("Failed to delete task")
							}
						}()
						mod.log.Info().Dur("ms", cfg.StartDelay).Msg("Waiting for task to run")
						select {
						case <-ctx.Done():
							mod.log.Warn().Msg("Cancelling execution")
							return err
						case <-time.After(cfg.StartDelay + (time.Second * 2)): // + two seconds
							// TODO: check if task is running yet; delete if the wait period is over
							break
						}
						return err
					}
				}
			}
		}
	} else if ecfg.ExecutionMethod == MethodDemand {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodDemandConfig); !ok {
			return errors.New("invalid configuration")

		} else {
			taskPath := cfg.TaskPath
			taskName := cfg.TaskName

			if taskName == "" {
				mod.log.Debug().Msg("Task name not defined. Using random string")
				taskName = util.RandomString()
			}
			if taskPath == "" {
				taskPath = `\` + taskName
			}
			if !TaskNameRegex.MatchString(taskName) {
				return fmt.Errorf("invalid task name: %s", taskName)
			}
			if !TaskPathRegex.MatchString(taskPath) {
				return fmt.Errorf("invalid task path: %s", taskPath)
			}

			mod.log.Debug().Msg("Using demand method")
			settings := defaultSettings
			settings.AllowStartOnDemand = true
			task := &task{
				TaskVersion:   "1.2",                                                   // static
				TaskNamespace: "http://schemas.microsoft.com/windows/2004/02/mit/task", // static
				Principals:    defaultPrincipals,
				Settings:      defaultSettings,
				Actions: actions{
					Context: defaultPrincipals.Principals[0].ID,
					Exec: []actionExec{
						{
							Command:   ecfg.ExecutableName,
							Arguments: ecfg.ExecutableArgs,
						},
					},
				},
			}
			if doc, err := xml.Marshal(task); err != nil {
				return fmt.Errorf("marshal task: %w", err)
			} else {
				docStr := TaskXMLHeader + string(doc)

				if err = mod.Connect(ctx, creds, target); err != nil {
					return fmt.Errorf("connect: %w", err)
				}
				defer func() {
					if err = mod.dce.Close(ctx); err != nil {
						mod.log.Warn().Err(err).Msg("Failed to dispose dce client")
					} else {
						mod.log.Debug().Msg("Disposed DCE client")
					}
				}()

				var response *itaskschedulerservice.RegisterTaskResponse
				if response, err = mod.tsch.RegisterTask(ctx, &itaskschedulerservice.RegisterTaskRequest{
					Path:       taskPath,
					XML:        docStr,
					Flags:      0, // TODO
					LogonType:  0, // TASK_LOGON_NONE
					CredsCount: 0,
					Creds:      nil,
				}); err != nil {
					return fmt.Errorf("register task: %w", err)

				} else {
					mod.log.Info().Str("task", response.ActualPath).Msg("Task registered successfully")
					if !cfg.NoDelete {
						defer func() {
							if err = mod.Cleanup(ctx, creds, target, &exec.CleanupConfig{
								CleanupMethod:       MethodDelete,
								CleanupMethodConfig: MethodDeleteConfig{TaskPath: taskPath},
							}); err != nil {
								mod.log.Error().Err(err).Msg("Failed to delete task")
							}
						}()
					}
					if _, err = mod.tsch.Run(ctx, &itaskschedulerservice.RunRequest{
						Path:  response.ActualPath,
						Flags: 0, // Maybe we want to use these?
					}); err != nil {
						return err
					} else {
						mod.log.Info().Str("task", response.ActualPath).Msg("Started task")
					}
				}
			}
		}
	} else {
		return fmt.Errorf("method not implemented: %s", ecfg.ExecutionMethod)
	}

	return nil
}
