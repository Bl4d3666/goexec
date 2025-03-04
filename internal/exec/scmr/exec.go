package scmrexec

import (
	"context"
	"errors"
	"fmt"
	dcerpc2 "github.com/FalconOpsLLC/goexec/internal/client/dcerpc"
	"github.com/FalconOpsLLC/goexec/internal/exec"
	"github.com/FalconOpsLLC/goexec/internal/windows"
	"github.com/RedTeamPentesting/adauth"
	"github.com/rs/zerolog"
)

const (
	MethodCreate string = "create"
	MethodModify string = "modify"

	ServiceModifyAccess uint32 = windows.SERVICE_QUERY_CONFIG | windows.SERVICE_CHANGE_CONFIG | windows.SERVICE_STOP | windows.SERVICE_START | windows.SERVICE_DELETE
	ServiceCreateAccess uint32 = windows.SC_MANAGER_CREATE_SERVICE | windows.SERVICE_START | windows.SERVICE_STOP | windows.SERVICE_DELETE
	ServiceAllAccess    uint32 = ServiceCreateAccess | ServiceModifyAccess
)

func (mod *Module) createClients(ctx context.Context) (cleanup func(cCtx context.Context), err error) {

	cleanup = func(context.Context) {
		if mod.dce != nil {
			mod.log.Debug().Msg("Cleaning up clients")
			if err := mod.dce.Close(ctx); err != nil {
				mod.log.Error().Err(err).Msg("Failed to destroy DCE connection")
			}
		}
	}
	cleanup(ctx)
	mod.dce = dcerpc2.NewDCEClient(ctx, false, &dcerpc2.SmbConfig{Port: 445})
	cleanup = func(context.Context) {}

	if err = mod.dce.Connect(ctx, mod.creds, mod.target); err != nil {
		return nil, fmt.Errorf("connection to DCERPC failed: %w", err)
	}
	mod.ctl, err = mod.dce.OpenSvcctl(ctx)
	return
}

func (mod *Module) Exec(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ecfg *exec.ExecutionConfig) (err error) {

	vctx := context.WithoutCancel(ctx)
	mod.log = zerolog.Ctx(ctx).With().
		Str("module", "scmr").
		Str("method", ecfg.ExecutionMethod).Logger()
	mod.creds = creds
	mod.target = target

	if ecfg.ExecutionMethod == MethodCreate {
		if cfg, ok := ecfg.ExecutionMethodConfig.(MethodCreateConfig); !ok || cfg.ServiceName == "" {
			return errors.New("invalid configuration")
		} else {
			if cleanup, err := mod.createClients(ctx); err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			} else {
				mod.log.Debug().Msg("Created clients")
				defer cleanup(ctx)
			}
			svc := &service{
				createConfig: &cfg,
				name:         cfg.ServiceName,
			}
			scm, code, err := mod.openSCM(ctx)
			if err != nil {
				return fmt.Errorf("failed to open SCM with code %d: %w", code, err)
			}
			mod.log.Debug().Msg("Opened handle to SCM")
			code, err = mod.createService(ctx, scm, svc, ecfg)
			if err != nil {
				return fmt.Errorf("failed to create service with code %d: %w", code, err)
			}
			mod.log.Info().Str("service", svc.name).Msg("Service created")
			// From here on out, make sure the service is properly deleted, even if the connection drops or something fails.
			if !cfg.NoDelete {
				defer func() {
					// TODO: stop service?
					if code, err = mod.deleteService(ctx, scm, svc); err != nil {
						mod.log.Error().Err(err).Msg("Failed to delete service") // TODO
					}
					mod.log.Info().Str("service", svc.name).Msg("Service deleted successfully")
				}()
			}
			if code, err = mod.startService(ctx, scm, svc); err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					// In case of timeout or cancel, try to reestablish a connection to restore the service
					mod.log.Info().Msg("Service start timeout/cancelled. Execution likely successful")
					mod.log.Info().Msg("Reconnecting for cleanup procedure")
					ctx = vctx

					if _, err = mod.createClients(ctx); err != nil {
						mod.log.Error().Err(err).Msg("Reconnect failed")

					} else if scm, code, err = mod.openSCM(ctx); err != nil {
						mod.log.Error().Err(err).Msg("Failed to reopen SCM")

					} else if svc.handle, code, err = mod.openService(ctx, scm, svc.name); err != nil {
						mod.log.Error().Str("service", svc.name).Err(err).Msg("Failed to reopen service handle")

					} else {
						mod.log.Debug().Str("service", svc.name).Msg("Reconnection successful")
					}
				} else {
					mod.log.Error().Err(err).Msg("Failed to start service")
				}
			} else {
				mod.log.Info().Str("service", svc.name).Msg("Execution successful")
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
			if cleanup, err := mod.createClients(ctx); err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			} else {
				mod.log.Debug().Msg("Created clients")
				defer cleanup(ctx)
			}
			svc := &service{modifyConfig: &cfg, name: cfg.ServiceName}

			// Open SCM handle
			scm, code, err := mod.openSCM(ctx)
			if err != nil {
				return fmt.Errorf("failed to create service with code %d: %w", code, err)
			}
			mod.log.Debug().Msg("Opened handle to SCM")

			// Open service handle
			if svc.handle, code, err = mod.openService(ctx, scm, svc.name); err != nil {
				return fmt.Errorf("failed to open service with code %d: %w", code, err)
			}
			mod.log.Debug().Str("service", svc.name).Msg("Opened service")

			// Stop service before editing
			if !cfg.NoStart {
				if code, err = mod.stopService(ctx, scm, svc); err != nil {
					mod.log.Warn().Err(err).Msg("Failed to stop existing service")
				} else if code == windows.ERROR_SERVICE_NOT_ACTIVE {
					mod.log.Debug().Str("service", svc.name).Msg("Service is not running")
				} else {
					mod.log.Info().Str("service", svc.name).Msg("Stopped existing service")
					defer func() {
						if code, err = mod.startService(ctx, scm, svc); err != nil {
							mod.log.Error().Err(err).Msg("Failed to restore service state to running")
						}
					}()
				}
			}
			if code, err = mod.queryServiceConfig(ctx, svc); err != nil {
				return fmt.Errorf("failed to query service configuration with code %d: %w", code, err)
			}
			mod.log.Debug().
				Str("service", svc.name).
				Str("command", svc.svcConfig.BinaryPathName).Msg("Fetched existing service configuration")

			// Change service configuration
			if code, err = mod.changeServiceConfigBinary(ctx, svc, cmd); err != nil {
				return fmt.Errorf("failed to edit service configuration with code %d: %w", code, err)
			}
			defer func() {
				// Revert configuration
				if code, err = mod.changeServiceConfigBinary(ctx, svc, svc.svcConfig.BinaryPathName); err != nil {
					mod.log.Error().Err(err).Msg("Failed to restore service configuration")
				} else {
					mod.log.Info().Str("service", svc.name).Msg("Restored service configuration")
				}
			}()
			mod.log.Info().
				Str("service", svc.name).
				Str("command", cmd).Msg("Changed service configuration")

			// Start service
			if !cfg.NoStart {
				if code, err = mod.startService(ctx, scm, svc); err != nil {
					if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
						// In case of timeout or cancel, try to reestablish a connection to restore the service
						mod.log.Info().Msg("Service start timeout/cancelled. Execution likely successful")
						mod.log.Info().Msg("Reconnecting for cleanup procedure")
						ctx = vctx

						if _, err = mod.createClients(ctx); err != nil {
							mod.log.Error().Err(err).Msg("Reconnect failed")

						} else if scm, code, err = mod.openSCM(ctx); err != nil {
							mod.log.Error().Err(err).Msg("Failed to reopen SCM")

						} else if svc.handle, code, err = mod.openService(ctx, scm, svc.name); err != nil {
							mod.log.Error().Str("service", svc.name).Err(err).Msg("Failed to reopen service handle")

						} else {
							mod.log.Debug().Str("service", svc.name).Msg("Reconnection successful")
						}
					} else {
						mod.log.Error().Err(err).Msg("Failed to start service")
					}
				} else {
					mod.log.Info().Str("service", svc.name).Msg("Started service")
				}
				defer func() {
					// Stop service
					if code, err = mod.stopService(ctx, scm, svc); err != nil {
						mod.log.Error().Err(err).Msg("Failed to stop service")
					} else {
						mod.log.Info().Str("service", svc.name).Msg("Stopped service")
					}
				}()
			}
		}
	} else {
		return fmt.Errorf("invalid method: %s", ecfg.ExecutionMethod)
	}
	return err
}
