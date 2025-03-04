package dcerpc

import (
	"context"
	"github.com/RedTeamPentesting/adauth"
)

type Client interface {
	Connect(ctx context.Context, creds *adauth.Credential, target *adauth.Target) error
	Close(ctx context.Context) error
}
