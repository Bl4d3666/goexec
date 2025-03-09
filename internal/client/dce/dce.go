package dce

import (
  "context"
  "fmt"
  "github.com/RedTeamPentesting/adauth"
  "github.com/RedTeamPentesting/adauth/dcerpcauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/epm/epm/v3"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/rs/zerolog"
)

var (
  NP   = "ncacn_np"
  TCP  = "ncacn_ip_tcp"
  HTTP = "ncacn_http"
)

type ConnectionMethodDCEConfig struct {
  NoEpm      bool                  // NoEpm disables EPM
  EpmAuto    bool                  // EpmAuto will find any suitable endpoint, without any filter
  Endpoint   *dcerpc.StringBinding // Endpoint is the endpoint passed to dcerpc.WithEndpoint. ignored if EpmAuto is false
  DceOptions []dcerpc.Option       // DceOptions are the options passed to dcerpc.Dial
  EpmOptions []dcerpc.Option       // EpmOptions are the options passed to epm.EndpointMapper
}

func (cfg *ConnectionMethodDCEConfig) GetDce(ctx context.Context, creds *adauth.Credential, target *adauth.Target, opts ...dcerpc.Option) (cc dcerpc.Conn, err error) {
  dceOpts := append(opts, cfg.DceOptions...)
  epmOpts := append(opts, cfg.EpmOptions...)

  log := zerolog.Ctx(ctx).With().
    Str("client", "DCERPC").Logger()

  // Mandatory logging
  dceOpts = append(dceOpts, dcerpc.WithLogger(log))
  epmOpts = append(epmOpts, dcerpc.WithLogger(log))

  ctx = gssapi.NewSecurityContext(ctx)
  ao, err := dcerpcauth.AuthenticationOptions(ctx, creds, target, &dcerpcauth.Options{})
  if err != nil {
    log.Error().Err(err).Msg("Failed to parse authentication options")
    return nil, fmt.Errorf("parse auth options: %w", err)
  }
  if cfg.Endpoint != nil && !cfg.EpmAuto {
    dceOpts = append(dceOpts, dcerpc.WithEndpoint(cfg.Endpoint.String()))
  }
  if !cfg.NoEpm {
    dceOpts = append(dceOpts,
      epm.EndpointMapper(ctx, target.AddressWithoutPort(), append(epmOpts, ao...)...))
  }
  return dcerpc.Dial(ctx, target.AddressWithoutPort(), append(dceOpts, ao...)...)
}
