package wmiexec

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/RedTeamPentesting/adauth"
  "github.com/RedTeamPentesting/adauth/dcerpcauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/iactivation/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemlevel1login/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemservices/v0"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/rs/zerolog"
)

const (
  ProtocolSequenceRPC uint16 = 7
  ProtocolSequenceNP  uint16 = 15
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
    Str("func", "Cleanup").Logger()

  if err = mod.dce.Close(ctx); err != nil {
    log.Warn().Err(err).Msg("Failed to close DCERPC connection")
  }
  return
}

func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, _ *exec.ConnectionConfig) (err error) {

  var baseOpts, authOpts []dcerpc.Option
  var ipid *dcom.IPID // This will store the IPID of the remote instance
  var bind2Opts []dcerpc.Option

  ctx = gssapi.NewSecurityContext(ctx)
  log := zerolog.Ctx(ctx).With().
    Str("func", "Connect").Logger()

  // Assemble DCERPC options
  {
    baseOpts = []dcerpc.Option{
      dcerpc.WithLogger(log),
      dcerpc.WithSign(), // Enforce signing
      dcerpc.WithSeal(), // Enforce packet stub encryption
    }
    // Add target name option if possible
    if tn, err := target.Hostname(ctx); err == nil {
      baseOpts = append(baseOpts, dcerpc.WithTargetName(tn))
    } else {
      log.Debug().Err(err).Msg("Failed to get target hostname")
    }
    // Parse target and credentials
    if authOpts, err = dcerpcauth.AuthenticationOptions(ctx, creds, target, &dcerpcauth.Options{}); err != nil {
      return fmt.Errorf("parse authentication options: %w", err)
    }
  }

  // Establish first connection (REMACT)
  {
    // Connection options
    rp := "ncacn_ip_tcp"                 // Underlying protocol
    ro := 135                            // RPC port
    rb := fmt.Sprintf("%s:[%d]", rp, ro) // RPC binding

    // Create DCERPC dialer
    mod.dce, err = dcerpc.Dial(ctx, target.AddressWithoutPort(), append(baseOpts, append(authOpts, dcerpc.WithEndpoint(rb))...)...)
    if err != nil {
      return fmt.Errorf("DCERPC dial: %w", err)
    }
    // Create remote activation client
    ia, err := iactivation.NewActivationClient(ctx, mod.dce, append(baseOpts, dcerpc.WithEndpoint(rb))...)
    if err != nil {
      return fmt.Errorf("create activation client: %w", err)
    }
    // Send remote activation request
    act, err := ia.RemoteActivation(ctx, &iactivation.RemoteActivationRequest{
      ORPCThis:                   ORPCThis,
      ClassID:                    wmi.Level1LoginClassID.GUID(),
      IIDs:                       []*dcom.IID{iwbemlevel1login.Level1LoginIID},
      RequestedProtocolSequences: []uint16{ProtocolSequenceRPC, ProtocolSequenceNP}, // TODO: dynamic
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
    ipid = act.InterfaceData[0].GetStandardObjectReference().Std.IPID

    // This ensures that the original target address/hostname is used in the string binding
    origBind := retBinds[0].String()
    if bind, err := dcerpc.ParseStringBinding(origBind); err != nil {
      log.Warn().Str("binding", origBind).Err(err).Msg("Failed to parse binding string returned by server")
      bind2Opts = act.OXIDBindings.EndpointsByProtocol(rp) // Try using the server supplied string binding
    } else {
      bind.NetworkAddress = target.AddressWithoutPort() // Replace address/hostname in new string binding
      bs := bind.String()
      log.Info().Str("binding", bs).Msg("found binding")
      bind2Opts = append(bind2Opts, dcerpc.WithEndpoint(bs)) // Use the new string binding
    }
  }

  // Establish second connection (WMI)
  {
    bind2Opts = append(bind2Opts, authOpts...)
    mod.dce, err = dcerpc.Dial(ctx, target.AddressWithoutPort(), append(baseOpts, bind2Opts...)...)
    if err != nil {
      return fmt.Errorf("dial WMI: %w", err)
    }
    // Create login client
    loginClient, err := iwbemlevel1login.NewLevel1LoginClient(ctx, mod.dce, append(baseOpts, dcom.WithIPID(ipid))...)
    if err != nil {
      return fmt.Errorf("initialize wbem login client: %w", err)
    }
    login, err := loginClient.NTLMLogin(ctx, &iwbemlevel1login.NTLMLoginRequest{ // TODO: Other login opts/methods?
      This:            ORPCThis,
      NetworkResource: "//./root/cimv2", // TODO: make this dynamic
    })
    if err != nil {
      return fmt.Errorf("ntlm login: %w", err)
    }

    mod.sc, err = iwbemservices.NewServicesClient(ctx, mod.dce, dcom.WithIPID(login.Namespace.InterfacePointer().IPID()))
    if err != nil {
      return fmt.Errorf("iwbemservices.NewServicesClient: %w", err)
    }
  }
  return nil
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {
  log := zerolog.Ctx(ctx).With().
    Str("module", "tsch").
    Str("method", ecfg.ExecutionMethod).Logger()

  if ecfg.ExecutionMethod == MethodCustom {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodCustomConfig); !ok {
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
