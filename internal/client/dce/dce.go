package dce

import (
  "context"
  "errors"
  "fmt"
  "github.com/RedTeamPentesting/adauth"
  "github.com/RedTeamPentesting/adauth/dcerpcauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/midl/uuid"
  "github.com/oiweiwei/go-msrpc/msrpc/epm/epm/v3"
  "github.com/oiweiwei/go-msrpc/ssp/gssapi"
  "github.com/rs/zerolog"
)

type ConnectionMethodDCEConfig struct {
  NoEpm      bool // NoEpm disables EPM
  EpmAuto    bool // EpmAuto will find any suitable endpoint, without any filter
  Insecure   bool
  NoSign     bool
  Endpoint   *dcerpc.StringBinding // Endpoint is the explicit endpoint passed to dcerpc.WithEndpoint for use without EPM
  EpmFilter  *dcerpc.StringBinding // EpmFilter is the rough filter used to pick an EPM endpoint
  DceOptions []dcerpc.Option       // DceOptions are the options passed to dcerpc.Dial
  EpmOptions []dcerpc.Option       // EpmOptions are the options passed to epm.EndpointMapper
}

func (cfg *ConnectionMethodDCEConfig) GetDce(ctx context.Context, cred *adauth.Credential, target *adauth.Target, endpoint, object string, opts ...dcerpc.Option) (cc dcerpc.Conn, err error) {
  dceOpts := append(opts, cfg.DceOptions...)
  epmOpts := append(opts, cfg.EpmOptions...)

  log := zerolog.Ctx(ctx).With().
    Str("client", "DCERPC").Logger()

  // Mandatory logging
  dceOpts = append(dceOpts, dcerpc.WithLogger(log))
  epmOpts = append(epmOpts, dcerpc.WithLogger(log))

  ctx = gssapi.NewSecurityContext(ctx)
  auth, err := dcerpcauth.AuthenticationOptions(ctx, cred, target, &dcerpcauth.Options{})
  if err != nil {
    log.Error().Err(err).Msg("Failed to parse authentication options")
    return nil, fmt.Errorf("parse auth options: %w", err)
  }
  addr := target.AddressWithoutPort()
  log = log.With().Str("address", addr).Logger()

  if object != "" {
    if id, err := uuid.Parse(object); err != nil {
      log.Error().Err(err).Msg("Failed to parse input object UUID")
    } else {
      dceOpts = append(dceOpts, dcerpc.WithObjectUUID(id))
    }
  }
  if cfg.Endpoint != nil {
    dceOpts = append(dceOpts, dcerpc.WithEndpoint(cfg.Endpoint.String()))
    log.Debug().Str("binding", cfg.Endpoint.String()).Msg("Using endpoint")

  } else if !cfg.NoEpm {
    dceOpts = append(dceOpts, epm.EndpointMapper(ctx, addr, append(epmOpts, auth...)...))
    log.Debug().Msg("Using endpoint mapper")

    if cfg.EpmFilter != nil {
      dceOpts = append(dceOpts, dcerpc.WithEndpoint(cfg.EpmFilter.String()))
      log.Debug().Str("filter", cfg.EpmFilter.String()).Msg("Using endpoint filter")
    }
  } else if endpoint != "" {
    dceOpts = append(dceOpts, dcerpc.WithEndpoint(endpoint))
    log.Debug().Str("endpoint", endpoint).Msg("Using default endpoint")

  } else {
    log.Err(err).Msg("Invalid DCE connection options")
    return nil, errors.New("get DCE: invalid connection options")
  }

  return dcerpc.Dial(ctx, target.AddressWithoutPort(), append(dceOpts, auth...)...)
}
