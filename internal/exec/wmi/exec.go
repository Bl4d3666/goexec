package wmiexec

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/RedTeamPentesting/adauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/iactivation/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemlevel1login/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemservices/v0"
  "github.com/rs/zerolog"
)

const (
  ProtocolSequenceRPC uint16 = 7
  ProtocolSequenceNP  uint16 = 15
  DefaultWmiEndpoint  string = "ncacn_ip_tcp:[135]"
)

var (
  ComVersion = &dcom.COMVersion{
    MajorVersion: 5,
    MinorVersion: 7,
  }
  ORPCThis = &dcom.ORPCThis{Version: ComVersion}
)

func (mod *Module) Cleanup(ctx context.Context, _ *exec.CleanupConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", "tsch").
    Str("func", "Cleanup").Logger()

  if err = mod.dce.Close(ctx); err != nil {
    log.Warn().Err(err).Msg("Failed to close DCERPC connection")
  }
  mod.sc = nil
  mod.dce = nil
  return
}

func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.ConnectionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("method", ccfg.ConnectionMethod).
    Str("func", "Connect").Logger()

  if cfg, ok := ccfg.ConnectionMethodConfig.(dce.ConnectionMethodDCEConfig); !ok {
    return errors.New("invalid configuration for DCE connection method")
  } else {
    var dceOpts []dcerpc.Option

    // Create DCE connection
    if mod.dce, err = cfg.GetDce(ctx, creds, target, DefaultWmiEndpoint, "", dceOpts...); err != nil {
      log.Error().Err(err).Msg("Failed to initialize DCE dialer")
      return fmt.Errorf("create DCE dialer: %w", err)
    }
    ia, err := iactivation.NewActivationClient(ctx, mod.dce)
    if err != nil {
      log.Error().Err(err).Msg("Failed to create activation client")
      return fmt.Errorf("create activation client: %w", err)
    }
    act, err := ia.RemoteActivation(ctx, &iactivation.RemoteActivationRequest{
      ORPCThis:                   ORPCThis,
      ClassID:                    wmi.Level1LoginClassID.GUID(),
      IIDs:                       []*dcom.IID{iwbemlevel1login.Level1LoginIID},
      RequestedProtocolSequences: []uint16{ProtocolSequenceRPC}, // TODO: Named pipe support
    })
    if err != nil {
      return fmt.Errorf("request remote activation: %w", err)
    }
    if act.HResult != 0 {
      return fmt.Errorf("remote activation failed with code %d", act.HResult)
    }
    retBinds := act.OXIDBindings.GetStringBindings()
    if len(act.InterfaceData) < 1 || len(retBinds) < 1 {
      return errors.New("remote activation failed")
    }
    ipid := act.InterfaceData[0].GetStandardObjectReference().Std.IPID

    for _, b := range retBinds {
      sb, err := dcerpc.ParseStringBinding("ncacn_ip_tcp:" + b.NetworkAddr)
      if err != nil {
        log.Debug().Err(err).Msg("Failed to parse string binding")
      }
      sb.NetworkAddress = target.AddressWithoutPort()
      dceOpts = append(dceOpts, dcerpc.WithEndpoint(sb.String()))
    }

    if mod.dce, err = cfg.GetDce(ctx, creds, target, DefaultWmiEndpoint, "", dceOpts...); err != nil {
      log.Error().Err(err).Msg("Failed to initialize secondary DCE dialer")
    }
    loginClient, err := iwbemlevel1login.NewLevel1LoginClient(ctx, mod.dce, dcom.WithIPID(ipid))
    if err != nil {
      return fmt.Errorf("initialize wbem login client: %w", err)
    }
    login, err := loginClient.NTLMLogin(ctx, &iwbemlevel1login.NTLMLoginRequest{
      This:            ORPCThis,
      NetworkResource: cfg.Resource,
    })
    if err != nil {
      return fmt.Errorf("ntlm login: %w", err)
    }

    mod.sc, err = iwbemservices.NewServicesClient(ctx, mod.dce, dcom.WithIPID(login.Namespace.InterfacePointer().IPID()))
    if err != nil {
      return fmt.Errorf("create services client: %w", err)
    }
  }
  return
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {
  log := zerolog.Ctx(ctx).With().
    Str("module", "tsch").
    Str("method", ecfg.ExecutionMethod).Logger()

  if ecfg.ExecutionMethod == MethodCall {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodCallConfig); !ok {
      return errors.New("invalid execution configuration")

    } else {
      out, err := mod.query(ctx, cfg.Class, cfg.Method, cfg.Arguments)
      if err != nil {
        return fmt.Errorf("query: %w", err)
      }
      if outJson, err := json.Marshal(out); err != nil {
        log.Error().Err(err).Msg("failed to marshal call output")
      } else {
        fmt.Println(string(outJson))
      }
    }
  } else if ecfg.ExecutionMethod == MethodProcess {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodProcessConfig); !ok {
      return errors.New("invalid execution configuration")
    } else {
      out, err := mod.query(ctx, "Win32_Process", "Create", map[string]any{
        "CommandLine": cfg.Command,
        "WorkingDir":  cfg.WorkingDirectory,
      })
      if err != nil {
        return fmt.Errorf("query: %w", err)
      }
      if pid, ok := out["ProcessId"]; ok && pid != nil {
        log.Info().
          Any("PID", pid).
          Any("return", out["ReturnValue"]).
          Msg("Process created")
      } else {
        log.Error().
          Any("return", out["ReturnValue"]).
          Msg("Process creation failed")
        return errors.New("failed to create process")
      }
    }
  } else {
    return errors.New("unsupported execution method")
  }
  return nil
}
