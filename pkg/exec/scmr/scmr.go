package scmrexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/FalconOpsLLC/goexec/internal/util"
	"github.com/FalconOpsLLC/goexec/pkg/exec"
	"github.com/FalconOpsLLC/goexec/pkg/windows"
)

type service struct {
	name         string
	exec         string
	createConfig *MethodCreateConfig
	modifyConfig *MethodModifyConfig

	svcState  uint32
	svcConfig *svcctl.QueryServiceConfigW
	handle    *svcctl.Handle
}

// openSCM opens a handle to SCM via ROpenSCManagerW
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/dc84adb3-d51d-48eb-820d-ba1c6ca5faf2
func (executor *Executor) openSCM(ctx context.Context) (scm *svcctl.Handle, code uint32, err error) {
	if executor.ctl != nil {

		hostname := executor.hostname
		if hostname == "" {
			hostname = util.RandomHostname()
		}
		if response, err := executor.ctl.OpenSCMW(ctx, &svcctl.OpenSCMWRequest{
			MachineName:   hostname + "\x00",    // lpMachineName; The server's name (i.e. DC01, dc01.domain.local)
			DatabaseName:  "ServicesActive\x00", // lpDatabaseName; must be "ServicesActive" or "ServicesFailed"
			DesiredAccess: ServiceModifyAccess,  // dwDesiredAccess; requested access - appears to be ignored?
		}); err != nil {

			if response != nil {
				return nil, response.Return, fmt.Errorf("open scm response: %w", err)
			}
			return nil, 0, fmt.Errorf("open scm: %w", err)
		} else {
			return response.SCM, 0, nil
		}
	}
	return nil, 0, errors.New("invalid arguments")
}

// createService creates a service with the provided configuration via RCreateServiceW
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/6a8ca926-9477-4dd4-b766-692fab07227e
func (executor *Executor) createService(ctx context.Context, scm *svcctl.Handle, scfg *service, ecfg *exec.ExecutionConfig) (code uint32, err error) {
	if executor.ctl != nil && scm != nil && scfg != nil && scfg.createConfig != nil {
		cfg := scfg.createConfig
		if response, err := executor.ctl.CreateServiceW(ctx, &svcctl.CreateServiceWRequest{
			ServiceManager: scm,
			ServiceName:    cfg.ServiceName + "\x00",
			DisplayName:    cfg.DisplayName + "\x00",
			BinaryPathName: ecfg.GetRawCommand() + "\x00",
			ServiceType:    cfg.ServiceType,
			StartType:      cfg.StartType,
			DesiredAccess:  ServiceCreateAccess,
		}); err != nil {

			if response != nil {
				return response.Return, fmt.Errorf("create service response: %w", err)
			}
			return 0, fmt.Errorf("create service: %w", err)
		} else {
			scfg.handle = response.Service
			return response.Return, err
		}
	}
	return 0, errors.New("invalid arguments")
}

// openService opens a handle to a service given the service name (lpServiceName)
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/6d0a4225-451b-4132-894d-7cef7aecfd2d
func (executor *Executor) openService(ctx context.Context, scm *svcctl.Handle, svcName string) (*svcctl.Handle, uint32, error) {
	if executor.ctl != nil && scm != nil {
		if openResponse, err := executor.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
			ServiceManager: scm,
			ServiceName:    svcName,
			DesiredAccess:  ServiceAllAccess,
		}); err != nil {
			if openResponse != nil {
				if openResponse.Return == windows.ERROR_SERVICE_DOES_NOT_EXIST {
					return nil, openResponse.Return, fmt.Errorf("remote service does not exist: %s", svcName)
				}
				return nil, openResponse.Return, fmt.Errorf("open service response: %w", err)
			}
			return nil, 0, fmt.Errorf("open service: %w", err)
		} else {
			return openResponse.Service, 0, nil
		}
	}
	return nil, 0, errors.New("invalid arguments")
}

// deleteService deletes an existing service with RDeleteService
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/6744cdb8-f162-4be0-bb31-98996b6495be
func (executor *Executor) deleteService(ctx context.Context, scm *svcctl.Handle, svc *service) (code uint32, err error) {
	if executor.ctl != nil && scm != nil && svc != nil {
		if deleteResponse, err := executor.ctl.DeleteService(ctx, &svcctl.DeleteServiceRequest{Service: svc.handle}); err != nil {
			defer func() {}()
			if deleteResponse != nil {
				return deleteResponse.Return, fmt.Errorf("delete service response: %w", err)
			}
			return 0, fmt.Errorf("delete service: %w", err)
		}
		return 0, nil
	}
	return 0, errors.New("invalid arguments")
}

// controlService sets the state of the provided process
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/e1c478be-117f-4512-9b67-17c20a48af97
func (executor *Executor) controlService(ctx context.Context, scm *svcctl.Handle, svc *service, control uint32) (code uint32, err error) {
	if executor.ctl != nil && scm != nil && svc != nil {
		if controlResponse, err := executor.ctl.ControlService(ctx, &svcctl.ControlServiceRequest{
			Service: svc.handle,
			Control: control,
		}); err != nil {
			if controlResponse != nil {
				return controlResponse.Return, fmt.Errorf("control service response: %w", err)
			}
			return 0, fmt.Errorf("control service: %w", err)
		}
		return 0, nil
	}
	return 0, errors.New("invalid arguments")
}

// stopService sends stop signal to existing service using controlService
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/e1c478be-117f-4512-9b67-17c20a48af97
func (executor *Executor) stopService(ctx context.Context, scm *svcctl.Handle, svc *service) (code uint32, err error) {
	if code, err = executor.controlService(ctx, scm, svc, windows.SERVICE_CONTROL_STOP); code == windows.ERROR_SERVICE_NOT_ACTIVE {
		err = nil
	}
	return
}

// startService starts the specified service with RStartServiceW
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/d9be95a2-cf01-4bdc-b30f-6fe4b37ada16
func (executor *Executor) startService(ctx context.Context, scm *svcctl.Handle, svc *service) (code uint32, err error) {
	if executor.ctl != nil && scm != nil && svc != nil {
		if startResponse, err := executor.ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: svc.handle}); err != nil {
			if startResponse != nil {
				// TODO: check if service is already running, return nil error if so
				if startResponse.Return == windows.ERROR_SERVICE_REQUEST_TIMEOUT {
					return 0, nil
				}
				return startResponse.Return, fmt.Errorf("start service response: %w", err)
			}
			return 0, fmt.Errorf("start service: %w", err)
		}
		return 0, nil
	}
	return 0, errors.New("invalid arguments")
}

// closeService closes the specified service handle using RCloseServiceHandle
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/a2a4e174-09fb-4e55-bad3-f77c4b13245c
func (executor *Executor) closeService(ctx context.Context, svc *svcctl.Handle) (code uint32, err error) {
	if executor.ctl != nil && svc != nil {
		if closeResponse, err := executor.ctl.CloseService(ctx, &svcctl.CloseServiceRequest{ServiceObject: svc}); err != nil {
			if closeResponse != nil {
				return closeResponse.Return, fmt.Errorf("close service response: %w", err)
			}
			return 0, fmt.Errorf("close service: %w", err)
		}
		return 0, nil
	}
	return 0, errors.New("invalid arguments")
}

// getServiceConfig fetches the configuration details of a service given a handle passed in a service{} structure.
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/89e2d5b1-19cf-44ca-969f-38eea9fe7f3c
func (executor *Executor) queryServiceConfig(ctx context.Context, svc *service) (code uint32, err error) {
	if executor.ctl != nil && svc != nil && svc.handle != nil {
		if getResponse, err := executor.ctl.QueryServiceConfigW(ctx, &svcctl.QueryServiceConfigWRequest{
			Service:      svc.handle,
			BufferLength: 1024 * 8,
		}); err != nil {
			if getResponse != nil {
				return getResponse.Return, fmt.Errorf("get service config response: %w", err)
			}
			return 0, fmt.Errorf("get service config: %w", err)
		} else {
			svc.svcConfig = getResponse.ServiceConfig
			return code, err
		}
	}
	return 0, errors.New("invalid arguments")
}

// queryServiceStatus fetches the state of the specified service
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/cf94d915-b4e1-40e5-872b-a9cb3ad09b46
func (executor *Executor) queryServiceStatus(ctx context.Context, svc *service) (uint32, error) {
	if executor.ctl != nil && svc != nil {
		if queryResponse, err := executor.ctl.QueryServiceStatus(ctx, &svcctl.QueryServiceStatusRequest{Service: svc.handle}); err != nil {
			if queryResponse != nil {
				return queryResponse.Return, fmt.Errorf("query service status response: %w", err)
			}
			return 0, fmt.Errorf("query service status: %w", err)
		} else {
			svc.svcState = queryResponse.ServiceStatus.CurrentState
			return 0, nil
		}
	}
	return 0, errors.New("invalid arguments")
}

// changeServiceConfigBinary edits the provided service's lpBinaryPathName
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-scmr/61ea7ed0-c49d-4152-a164-b4830f16c8a4
func (executor *Executor) changeServiceConfigBinary(ctx context.Context, svc *service, bin string) (code uint32, err error) {
	if executor.ctl != nil && svc != nil && svc.handle != nil {
		if changeResponse, err := executor.ctl.ChangeServiceConfigW(ctx, &svcctl.ChangeServiceConfigWRequest{
			Service:        svc.handle,
			ServiceType:    svc.svcConfig.ServiceType,
			StartType:      svc.svcConfig.StartType,
			ErrorControl:   svc.svcConfig.ErrorControl,
			BinaryPathName: bin + "\x00",
			LoadOrderGroup: svc.svcConfig.LoadOrderGroup,
			TagID:          svc.svcConfig.TagID,
			// Dependencies: svc.svcConfig.Dependencies // TODO
			ServiceStartName: svc.svcConfig.ServiceStartName,
			DisplayName:      svc.svcConfig.DisplayName,
		}); err != nil {
			if changeResponse != nil {
				return changeResponse.Return, fmt.Errorf("change service config response: %w", err)
			}
			return 0, fmt.Errorf("change service config: %w", err)
		}
		return
	}
	return 0, errors.New("invalid arguments")
}
