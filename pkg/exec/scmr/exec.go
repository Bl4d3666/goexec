package scmrexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/bryanmcnulty/adauth"
	"github.com/rs/zerolog"
	"github.com/FalconOpsLLC/goexec/pkg/client/dcerpc"
	"github.com/FalconOpsLLC/goexec/pkg/exec"
	"github.com/FalconOpsLLC/goexec/pkg/windows"
)

const (
	MethodCreate string = "create"
	MethodModify string = "modify"

	ServiceModifyAccess uint32 = windows.SERVICE_QUERY_CONFIG | windows.SERVICE_CHANGE_CONFIG | windows.SERVICE_STOP | windows.SERVICE_START | windows.SERVICE_DELETE
	ServiceCreateAccess uint32 = windows.SC_MANAGER_CREATE_SERVICE | windows.SERVICE_START | windows.SERVICE_STOP | windows.SERVICE_DELETE
	ServiceAllAccess    uint32 = ServiceCreateAccess | ServiceModifyAccess
)

func (executor *Executor) createClients(ctx context.Context) (cleanup func(cCtx context.Context), err error) {

	cleanup = func(context.Context) {
		if executor.dce != nil {
			executor.log.Debug().Msg("Cleaning up clients")
			if err := executor.dce.Close(ctx); err != nil {
				executor.log.Error().Err(err).Msg("Failed to destroy DCE connection")
			}
		}
	}
	cleanup(ctx)
	executor.dce = dcerpc.NewDCEClient(ctx, false, &dcerpc.SmbConfig{Port: 445})
	cleanup = func(context.Context) {}

	if err = executor.dce.Connect(ctx, executor.creds, executor.target); err != nil {
		return nil, fmt.Errorf("connection to DCERPC failed: %w", err)
	}
	executor.ctl, err = executor.dce.OpenSvcctl(ctx)
	return
}

func (executor *Executor) Exec(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ecfg *exec.ExecutionConfig) (err error) {

	vctx := context.WithoutCancel(ctx)
	executor.log = zerolog.Ctx(ctx).With().
		Str("module", "scmr").
		Str("method", ecfg.ExecutionMethod).Logger()
	executor.creds = creds
	executor.target = target

	if ecfg.ExecutionMethod == MethodCreate {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodCreateConfig); !ok || cfg.ServiceName == "" {
			return errors.New("invalid configuration")
		} else {
			if cleanup, err := executor.createClients(ctx); err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			} else {
				executor.log.Debug().Msg("Created clients")
				defer cleanup(ctx)
			}
			svc := &service{
				createConfig: &cfg,
				name:         cfg.ServiceName,
			}
			scm, code, err := executor.openSCM(ctx)
			if err != nil {
				return fmt.Errorf("failed to open SCM with code %d: %w", code, err)
			}
			executor.log.Debug().Msg("Opened handle to SCM")
			code, err = executor.createService(ctx, scm, svc, ecfg)
			if err != nil {
				return fmt.Errorf("failed to create service with code %d: %w", code, err)
			}
			executor.log.Info().Str("service", svc.name).Msg("Service created")
			// From here on out, make sure the service is properly deleted, even if the connection drops or something fails.
			if !cfg.NoDelete {
				defer func() {
					// TODO: stop service?
					if code, err = executor.deleteService(ctx, scm, svc); err != nil {
						executor.log.Error().Err(err).Msg("Failed to delete service") // TODO
					}
					executor.log.Info().Str("service", svc.name).Msg("Service deleted successfully")
				}()
			}
			if code, err = executor.startService(ctx, scm, svc); err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					// In case of timeout or cancel, try to reestablish a connection to restore the service
					executor.log.Info().Msg("Service start timeout/cancelled. Execution likely successful")
					executor.log.Info().Msg("Reconnecting for cleanup procedure")
					ctx = vctx

					if _, err = executor.createClients(ctx); err != nil {
						executor.log.Error().Err(err).Msg("Reconnect failed")

					} else if scm, code, err = executor.openSCM(ctx); err != nil {
						executor.log.Error().Err(err).Msg("Failed to reopen SCM")

					} else if svc.handle, code, err = executor.openService(ctx, scm, svc.name); err != nil {
						executor.log.Error().Str("service", svc.name).Err(err).Msg("Failed to reopen service handle")

					} else {
						executor.log.Debug().Str("service", svc.name).Msg("Reconnection successful")
					}
				} else {
					executor.log.Error().Err(err).Msg("Failed to start service")
				}
			} else {
				executor.log.Info().Str("service", svc.name).Msg("Execution successful")
			}
		}
	} else if ecfg.ExecutionMethod == MethodModify {
		// Use service modification method
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodModifyConfig); !ok || cfg.ServiceName == "" {
			return errors.New("invalid configuration")

		} else {
			// Ensure that a command (executable full path + args) is supplied
			cmd := ecfg.GetRawCommand()
			if cmd == "" {
				return errors.New("no command provided")
			}

			// Initialize protocol clients
			if cleanup, err := executor.createClients(ctx); err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			} else {
				executor.log.Debug().Msg("Created clients")
				defer cleanup(ctx)
			}
			svc := &service{modifyConfig: &cfg, name: cfg.ServiceName}

			// Open SCM handle
			scm, code, err := executor.openSCM(ctx)
			if err != nil {
				return fmt.Errorf("failed to create service with code %d: %w", code, err)
			}
			executor.log.Debug().Msg("Opened handle to SCM")

			// Open service handle
			if svc.handle, code, err = executor.openService(ctx, scm, svc.name); err != nil {
				return fmt.Errorf("failed to open service with code %d: %w", code, err)
			}
			executor.log.Debug().Str("service", svc.name).Msg("Opened service")

			// Stop service before editing
			if !cfg.NoStart {
				if code, err = executor.stopService(ctx, scm, svc); err != nil {
					executor.log.Warn().Err(err).Msg("Failed to stop existing service")
				} else if code == windows.ERROR_SERVICE_NOT_ACTIVE {
					executor.log.Debug().Str("service", svc.name).Msg("Service is not running")
				} else {
					executor.log.Info().Str("service", svc.name).Msg("Stopped existing service")
					defer func() {
						if code, err = executor.startService(ctx, scm, svc); err != nil {
							executor.log.Error().Err(err).Msg("Failed to restore service state to running")
						}
					}()
				}
			}
			if code, err = executor.queryServiceConfig(ctx, svc); err != nil {
				return fmt.Errorf("failed to query service configuration with code %d: %w", code, err)
			}
			executor.log.Debug().
				Str("service", svc.name).
				Str("command", svc.svcConfig.BinaryPathName).Msg("Fetched existing service configuration")

			// Change service configuration
			if code, err = executor.changeServiceConfigBinary(ctx, svc, cmd); err != nil {
				return fmt.Errorf("failed to edit service configuration with code %d: %w", code, err)
			}
			defer func() {
				// Revert configuration
				if code, err = executor.changeServiceConfigBinary(ctx, svc, svc.svcConfig.BinaryPathName); err != nil {
					executor.log.Error().Err(err).Msg("Failed to restore service configuration")
				} else {
					executor.log.Info().Str("service", svc.name).Msg("Restored service configuration")
				}
			}()
			executor.log.Info().
				Str("service", svc.name).
				Str("command", cmd).Msg("Changed service configuration")

			// Start service
			if !cfg.NoStart {
				if code, err = executor.startService(ctx, scm, svc); err != nil {
					if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
						// In case of timeout or cancel, try to reestablish a connection to restore the service
						executor.log.Info().Msg("Service start timeout/cancelled. Execution likely successful")
						executor.log.Info().Msg("Reconnecting for cleanup procedure")
						ctx = vctx

						if _, err = executor.createClients(ctx); err != nil {
							executor.log.Error().Err(err).Msg("Reconnect failed")

						} else if scm, code, err = executor.openSCM(ctx); err != nil {
							executor.log.Error().Err(err).Msg("Failed to reopen SCM")

						} else if svc.handle, code, err = executor.openService(ctx, scm, svc.name); err != nil {
							executor.log.Error().Str("service", svc.name).Err(err).Msg("Failed to reopen service handle")

						} else {
							executor.log.Debug().Str("service", svc.name).Msg("Reconnection successful")
						}
					} else {
						executor.log.Error().Err(err).Msg("Failed to start service")
					}
				} else {
					executor.log.Info().Str("service", svc.name).Msg("Started service")
				}
				defer func() {
					// Stop service
					if code, err = executor.stopService(ctx, scm, svc); err != nil {
						executor.log.Error().Err(err).Msg("Failed to stop service")
					} else {
						executor.log.Info().Str("service", svc.name).Msg("Stopped service")
					}
				}()
			}
		}
	}
	return err
}
