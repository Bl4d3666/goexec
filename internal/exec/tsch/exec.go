package tschexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/FalconOpsLLC/goexec/internal/client/dce"
	"github.com/FalconOpsLLC/goexec/internal/exec"
	"github.com/FalconOpsLLC/goexec/internal/util"
	"github.com/RedTeamPentesting/adauth"
	"github.com/RedTeamPentesting/adauth/dcerpcauth"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/midl/uuid"
	"github.com/oiweiwei/go-msrpc/msrpc/epm/epm/v3"
	"github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
	"github.com/oiweiwei/go-msrpc/ssp/gssapi"
	"github.com/rs/zerolog"
	"time"
)

const (
	DefaultEndpoint = "ncacn_np:[atsvc]"
)

var (
	TschRpcUuid                = uuid.MustParse("86D35949-83C9-4044-B424-DB363231FD0C")
	SupportedEndpointProtocols = []string{"ncacn_np", "ncacn_ip_tcp"}
)

// Connect to the target & initialize DCE & TSCH clients
func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.ConnectionConfig) (err error) {

	//var port uint16
	var endpoint string = DefaultEndpoint
	var epmOpts []dcerpc.Option
	var dceOpts []dcerpc.Option

	log := zerolog.Ctx(ctx).With().
		Str("func", "Connect").Logger()

	if mod.dce == nil {
		if ccfg.ConnectionMethod == exec.ConnectionMethodDCE {
			if cfg, ok := ccfg.ConnectionMethodConfig.(dce.ConnectionMethodDCEConfig); !ok {
				return fmt.Errorf("invalid configuration for DCE connection method")
			} else {
				// Connect to ITaskSchedulerService
				{
					// Parse target & creds
					ctx = gssapi.NewSecurityContext(ctx)
					ao, err := dcerpcauth.AuthenticationOptions(ctx, creds, target, &dcerpcauth.Options{})
					if err != nil {
						log.Error().Err(err).Msg("Failed to parse authentication options")
						return fmt.Errorf("parse auth options: %w", err)
					}
					dceOpts = append(cfg.Options,
						dcerpc.WithLogger(log),
						dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy), // AuthLevelPktPrivacy is required for TSCH/ATSVC
						dcerpc.WithObjectUUID(TschRpcUuid))

					if cfg.Endpoint != nil {
						endpoint = cfg.Endpoint.String()
					}
					if cfg.NoEpm {
						dceOpts = append(dceOpts, dcerpc.WithEndpoint(endpoint))
					} else {
						epmOpts = append(epmOpts, dceOpts...)
						dceOpts = append(dceOpts,
							epm.EndpointMapper(ctx, target.AddressWithoutPort(), append(epmOpts, ao...)...))
						if !cfg.EpmAuto {
							dceOpts = append(dceOpts, dcerpc.WithEndpoint(endpoint))
						}
					}
					log = log.With().Str("endpoint", endpoint).Logger()
					log.Info().Msg("Connecting to target")

					// Create DCERPC dialer
					mod.dce, err = dcerpc.Dial(ctx, target.AddressWithoutPort(), append(dceOpts, ao...)...)
					if err != nil {
						log.Error().Err(err).Msg("Failed to create DCERPC dialer")
						return fmt.Errorf("create DCERPC dialer: %w", err)
					}

					// Create ITaskSchedulerService
					mod.tsch, err = itaskschedulerservice.NewTaskSchedulerServiceClient(ctx, mod.dce)
					if err != nil {
						log.Error().Err(err).Msg("Failed to initialize TSCH client")
						return fmt.Errorf("init TSCH client: %w", err)
					}
					log.Info().Msg("DCE connection successful")
				}
			}
		} else {
			return errors.New("unsupported connection method")
		}
	}
	return
}

func (mod *Module) Cleanup(ctx context.Context, ccfg *exec.CleanupConfig) (err error) {
	log := zerolog.Ctx(ctx).With().
		Str("method", ccfg.CleanupMethod).
		Str("func", "Cleanup").Logger()

	if ccfg.CleanupMethod == MethodDelete {
		if cfg, ok := ccfg.CleanupMethodConfig.(MethodDeleteConfig); !ok {
			return errors.New("invalid configuration")

		} else {
			log = log.With().Str("task", cfg.TaskPath).Logger()
			log.Info().Msg("Manually deleting task")

			if err = mod.deleteTask(ctx, cfg.TaskPath); err == nil {
				log.Info().Msg("Task deleted successfully")
			}
		}
	} else if ccfg.CleanupMethod == "" {
		return nil
	} else {
		return fmt.Errorf("unsupported cleanup method")
	}
	return
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {

	log := zerolog.Ctx(ctx).With().
		Str("method", ecfg.ExecutionMethod).
		Str("func", "Exec").Logger()

	if ecfg.ExecutionMethod == MethodRegister {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodRegisterConfig); !ok {
			return errors.New("invalid configuration")

		} else {
			startTime := time.Now().UTC().Add(cfg.StartDelay)
			stopTime := startTime.Add(cfg.StopDelay)

			tr := taskTimeTrigger{
				StartBoundary: startTime.Format(TaskXMLDurationFormat),
				//EndBoundary:   stopTime.Format(TaskXMLDurationFormat),
				Enabled: true,
			}
			tk := newTask(nil, nil, triggers{TimeTriggers: []taskTimeTrigger{tr}}, ecfg.ExecutableName, ecfg.ExecutableArgs)

			if !cfg.NoDelete && !cfg.CallDelete {
				if cfg.StopDelay == 0 {
					cfg.StopDelay = time.Second
				}
				tk.Settings.DeleteExpiredTaskAfter = xmlDuration(cfg.DeleteDelay)
				tk.Triggers.TimeTriggers[0].EndBoundary = stopTime.Format(TaskXMLDurationFormat)
			}
			taskPath := cfg.TaskPath
			if taskPath == "" {
				log.Debug().Msg("Task path not defined. Using random path")
				taskPath = `\` + util.RandomString()
			}
			// The taskPath is changed here to the *actual path returned by SchRpcRegisterTask
			taskPath, err = mod.registerTask(ctx, *tk, taskPath)
			if err != nil {
				return fmt.Errorf("call registerTask: %w", err)
			}

			if !cfg.NoDelete {
				if cfg.CallDelete {
					defer mod.deleteTask(ctx, taskPath)

					log.Info().Dur("ms", cfg.StartDelay).Msg("Waiting for task to run")
					select {
					case <-ctx.Done():
						log.Warn().Msg("Cancelling execution")
						return err
					case <-time.After(cfg.StartDelay + time.Second): // + one second for good measure
						for {
							if stat, err := mod.tsch.GetLastRunInfo(ctx, &itaskschedulerservice.GetLastRunInfoRequest{Path: taskPath}); err != nil {
								log.Warn().Err(err).Msg("Failed to get last run info. Assuming task was executed")
								break
							} else if stat.LastRuntime.AsTime().IsZero() {
								log.Warn().Msg("Task was not yet run. Waiting 10 additional seconds")
								time.Sleep(10 * time.Second)
							} else {
								break
							}
						}
						break
					}
				} else {
					log.Info().Time("when", stopTime).Msg("Task is scheduled to delete")
				}
			}
		}
	} else if ecfg.ExecutionMethod == MethodDemand {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodDemandConfig); !ok {
			return errors.New("invalid configuration")

		} else {
			taskPath := cfg.TaskPath
			if taskPath == "" {
				log.Debug().Msg("Task path not defined. Using random path")
				taskPath = `\` + util.RandomString()
			}
			st := newSettings(true, true, false)
			tk := newTask(st, nil, triggers{}, ecfg.ExecutableName, ecfg.ExecutableArgs)

			// The taskPath is changed here to the *actual path returned by SchRpcRegisterTask
			taskPath, err = mod.registerTask(ctx, *tk, taskPath)
			if err != nil {
				return fmt.Errorf("call registerTask: %w", err)
			}
			if !cfg.NoDelete {
				defer mod.deleteTask(ctx, taskPath)
			}
			_, err := mod.tsch.Run(ctx, &itaskschedulerservice.RunRequest{
				Path:  taskPath,
				Flags: 0, // Maybe we want to use these?
			})
			if err != nil {
				log.Error().Str("task", taskPath).Err(err).Msg("Failed to run task")
				return fmt.Errorf("force run task: %w", err)
			}
			log.Info().Str("task", taskPath).Msg("Started task")
		}
	} else {
		return fmt.Errorf("method '%s' not implemented", ecfg.ExecutionMethod)
	}

	return nil
}
