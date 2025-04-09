package dcomexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/internal/client/dce"
  "github.com/FalconOpsLLC/goexec/internal/exec"
  "github.com/RedTeamPentesting/adauth"
  guuid "github.com/google/uuid"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/midl/uuid"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/iremotescmactivator/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dtyp"
  "github.com/rs/zerolog"
)

const (
  DefaultDcomEndpoint = "ncacn_ip_tcp:[135]"
)

var (
  MmcUuid          = uuid.MustParse("49B2791A-B1AE-4C90-9B8E-E860BA07F889")
  ShellWindowsUuid = uuid.MustParse("9BA05972-F6A8-11CF-A442-00A0C90A8F39")
  RandCid          = dcom.CID(*dtyp.GUIDFromUUID(uuid.MustParse(guuid.NewString())))
  IDispatchIID     = &dcom.IID{
    Data1: 0x20400,
    Data2: 0x0,
    Data3: 0x0,
    Data4: []byte{0xc0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x46},
  }
  ComVersion = &dcom.COMVersion{
    MajorVersion: 5,
    MinorVersion: 7,
  }
  MmcClsid = dcom.ClassID(*dtyp.GUIDFromUUID(MmcUuid))
  ORPCThis = &dcom.ORPCThis{
    Version: ComVersion,
    CID:     &RandCid,
  }
)

func (mod *Module) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, ccfg *exec.ConnectionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("method", ccfg.ConnectionMethod).
    Str("func", "Exec").Logger()

  if ccfg.ConnectionMethod == exec.ConnectionMethodDCE {
    if cfg, ok := ccfg.ConnectionMethodConfig.(dce.ConnectionMethodDCEConfig); !ok {
      return errors.New("invalid configuration for DCE connection method")
    } else {
      opts := []dcerpc.Option{dcerpc.WithSign(), dcerpc.WithSecurityLevel(0)}

      // Create DCE connection
      if mod.dce, err = cfg.GetDce(ctx, creds, target, DefaultDcomEndpoint, "", opts...); err != nil {
        log.Error().Err(err).Msg("Failed to initialize DCE dialer")
        return fmt.Errorf("create DCE dialer: %w", err)
      }

      inst := &dcom.InstantiationInfoData{
        ClassID:          &MmcClsid,
        IID:              []*dcom.IID{IDispatchIID},
        ClientCOMVersion: ComVersion,
      }
      scm := &dcom.SCMRequestInfoData{
        RemoteRequest: &dcom.CustomRemoteRequestSCMInfo{
          RequestedProtocolSequences: []uint16{7},
        },
      }
      loc := &dcom.LocationInfoData{}
      ac := &dcom.ActivationContextInfoData{}
      ap := &dcom.ActivationProperties{
        DestinationContext: 2,
        Properties:         []dcom.ActivationProperty{inst, ac, loc, scm},
      }
      apin, err := ap.ActivationPropertiesIn()
      if err != nil {
        return err
      }
      act, err := iremotescmactivator.NewRemoteSCMActivatorClient(ctx, mod.dce)
      if err != nil {
        return err
      }
      cr, err := act.RemoteCreateInstance(ctx, &iremotescmactivator.RemoteCreateInstanceRequest{
        ORPCThis: &dcom.ORPCThis{
          Version: ComVersion,
          Flags:   1,
          CID:     &RandCid,
        },
        ActPropertiesIn: apin,
      })
      if err != nil {
        return err
      }
      log.Info().Msg("RemoteCreateInstance succeeded")

      apout := &dcom.ActivationProperties{}
      if err = apout.Parse(cr.ActPropertiesOut); err != nil {
        return err
      }
      si := apout.SCMReplyInfoData()
      pi := apout.PropertiesOutInfo()

      if si == nil {
        return fmt.Errorf("remote create instance response: SCMReplyInfoData is nil")
      }
      if pi == nil {
        return fmt.Errorf("remote create instance response: PropertiesOutInfo is nil")
      }
      oIPID := pi.InterfaceData[0].IPID()

      opts = append(opts, si.RemoteReply.OXIDBindings.EndpointsByProtocol("ncacn_ip_tcp")...) // TODO
      mod.dce, err = cfg.GetDce(ctx, creds, target, DefaultDcomEndpoint, "", opts...)
      if err != nil {
        return err
      }
      log.Info().Msg("created new DCERPC dialer")

      mod.dc, err = idispatch.NewDispatchClient(ctx, mod.dce, dcom.WithIPID(oIPID))
      if err != nil {
        return err
      }
      log.Info().Msg("created IDispatch client")
    }
  }
  return
}

func (mod *Module) Exec(ctx context.Context, ecfg *exec.ExecutionConfig) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("method", ecfg.ExecutionMethod).
    Str("func", "Exec").Logger()

  if ecfg.ExecutionMethod == MethodMmc {
    if cfg, ok := ecfg.ExecutionMethodConfig.(MethodMmcConfig); !ok {
      return errors.New("invalid configuration")

    } else {
      // https://learn.microsoft.com/en-us/previous-versions/windows/desktop/mmc/view-executeshellcommand
      method := "Document.ActiveView.ExecuteShellCommand"
      log = log.With().Str("classMethod", method).Logger()

      log.Info().
        Str("executable", ecfg.ExecutableName).
        Str("arguments", ecfg.ExecutableArgs).Msg("Attempting execution")

      command := &oaut.Variant{
        Size:     5,
        VT:       8,
        VarUnion: &oaut.Variant_VarUnion{Value: &oaut.Variant_VarUnion_BSTR{BSTR: &oaut.String{Data: ecfg.ExecutableName}}},
      }
      directory := &oaut.Variant{
        Size:     5,
        VT:       8,
        VarUnion: &oaut.Variant_VarUnion{Value: &oaut.Variant_VarUnion_BSTR{BSTR: &oaut.String{Data: cfg.WorkingDirectory}}},
      }
      parameters := &oaut.Variant{
        Size:     5,
        VT:       8,
        VarUnion: &oaut.Variant_VarUnion{Value: &oaut.Variant_VarUnion_BSTR{BSTR: &oaut.String{Data: ecfg.ExecutableArgs}}},
      }
      windowState := &oaut.Variant{
        Size:     5,
        VT:       8,
        VarUnion: &oaut.Variant_VarUnion{Value: &oaut.Variant_VarUnion_BSTR{BSTR: &oaut.String{Data: cfg.WindowState}}},
      }
      // Arguments must be passed in reverse order
      if _, err := callMethod(ctx, mod.dc, method, windowState, parameters, directory, command); err != nil {
        log.Error().Err(err).Msg("Failed to call method")
        return fmt.Errorf("call %q: %w", method, err)
      }
      log.Info().Msg("Method call successful")
    }
  }
  return nil
}
