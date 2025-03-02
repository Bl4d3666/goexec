package dcerpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/bryanmcnulty/adauth"
	"github.com/bryanmcnulty/adauth/dcerpcauth"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/epm/epm/v3"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/oiweiwei/go-msrpc/smb2"
	"github.com/oiweiwei/go-msrpc/ssp/gssapi"
	"github.com/rs/zerolog"
)

const (
	DceDefaultProto string = "ncacn_np"
	DceDefaultPort  uint16 = 445
)

type DCEClient struct {
	Port  uint16
	Proto string

	log      zerolog.Logger
	conn     dcerpc.Conn
	opts     []dcerpc.Option
	authOpts dcerpcauth.Options
}

func NewDCEClient(ctx context.Context, insecure bool, smbConfig *SmbConfig) (client *DCEClient) {
	client = &DCEClient{
		Port:     DceDefaultPort,
		Proto:    DceDefaultProto,
		log:      zerolog.Ctx(ctx).With().Str("client", "DCE").Logger(),
		authOpts: dcerpcauth.Options{},
	}
	client.opts = []dcerpc.Option{dcerpc.WithLogger(client.log)}
	client.authOpts = dcerpcauth.Options{Debug: client.log.Trace().Msgf}

	if smbConfig != nil {
		if smbConfig.Port != 0 {
			client.Port = smbConfig.Port
			client.opts = append(client.opts, dcerpc.WithSMBPort(int(smbConfig.Port)))
		}
	}
	if insecure {
		client.log.Debug().Msg("Using insecure DCERPC connection")
		client.opts = append(client.opts, dcerpc.WithInsecure())
	} else {
		client.log.Debug().Msg("Using secure DCERPC connection")
		client.authOpts.SMBOptions = append(client.authOpts.SMBOptions, smb2.WithSeal())
	}
	return
}

func (client *DCEClient) OpenSvcctl(ctx context.Context) (ctl svcctl.SvcctlClient, err error) {
	if client.conn == nil {
		return nil, errors.New("DCE connection not open")
	}
	if ctl, err = svcctl.NewSvcctlClient(ctx, client.conn, dcerpc.WithInsecure()); err != nil {
		client.log.Debug().Err(err).Msg("Failed to open Svcctl client")
	}
	return
}

func (client *DCEClient) DCE() dcerpc.Conn {
	return client.conn
}

func (client *DCEClient) Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target, dialOpts ...dcerpc.Option) (err error) {
	if creds != nil && target != nil {
		authCtx := gssapi.NewSecurityContext(ctx)

		binding := fmt.Sprintf(`%s:%s`, client.Proto, target.AddressWithoutPort())
		mapper := epm.EndpointMapper(ctx, fmt.Sprintf("%s:%d", target.AddressWithoutPort(), client.Port), dcerpc.WithLogger(client.log))
		dceOpts := []dcerpc.Option{dcerpc.WithLogger(client.log), dcerpc.WithSeal()}

		if dceOpts, err = dcerpcauth.AuthenticationOptions(authCtx, creds, target, &client.authOpts); err == nil {
			dceOpts = append(dceOpts, mapper)
			dceOpts = append(dceOpts, client.opts...)
			dceOpts = append(dceOpts, dialOpts...)

			if client.conn, err = dcerpc.Dial(authCtx, binding, dceOpts...); err == nil {
				client.log.Debug().Msg("Bind successful")
				return nil
			}
			client.log.Debug().Err(err).Msg("DCERPC bind failed")
			return errors.New("bind failed")
		}
		return errors.New("unable to parse DCE authentication options")
	}
	return errors.New("invalid arguments")
}

func (client *DCEClient) Close(ctx context.Context) (err error) {
	if client.conn == nil {
		client.log.Debug().Msg("Connection already closed")
	} else if err = client.conn.Close(ctx); err == nil {
		client.log.Debug().Msg("Connection closed successfully")
	} else {
		client.log.Error().Err(err).Msg("Failed to close connection")
	}
	return
}
