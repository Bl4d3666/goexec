package dce

import (
  "context"
  "fmt"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/epm/epm/v3"
  "github.com/rs/zerolog"
)

type Client struct {
  Options

  conn     dcerpc.Conn
  hostname string
}

func NewClient() *Client {
  return new(Client)
}

func (c *Client) String() string {
  return ClientName
}

func (c *Client) Reconnect(ctx context.Context, opts ...dcerpc.Option) (err error) {
  c.DcerpcOptions = append(c.DcerpcOptions, opts...)

  return c.Connect(ctx)
}

func (c *Client) Dce() (dce dcerpc.Conn) {
  return c.conn
}

func (c *Client) Logger(ctx context.Context) (log zerolog.Logger) {
  return zerolog.Ctx(ctx).With().
    Str("client", c.String()).Logger()
}

func (c *Client) Connect(ctx context.Context) (err error) {

  log := c.Logger(ctx)
  ctx = log.WithContext(ctx)

  var do, eo []dcerpc.Option

  do = append(do, c.DcerpcOptions...)
  do = append(do, c.authOptions...)

  if !c.NoSign {
    do = append(do, dcerpc.WithSign())
    eo = append(eo, dcerpc.WithSign())
  }
  if !c.NoSeal {
    do = append(do, dcerpc.WithSeal(), dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy))
    eo = append(eo, dcerpc.WithSeal(), dcerpc.WithSecurityLevel(dcerpc.AuthLevelPktPrivacy))
  }

  if !c.NoLog {
    do = append(do, dcerpc.WithLogger(log))
    eo = append(eo, dcerpc.WithLogger(log))
  }

  if !c.NoEpm {
    log.Debug().Msg("Using endpoint mapper")

    eo = append(eo, c.epmOptions...)
    eo = append(eo, c.authOptions...)

    do = append(do, epm.EndpointMapper(ctx, c.Host, eo...))
  }

  for _, e := range c.stringBindings {
    do = append(do, dcerpc.WithEndpoint(e.String()))
  }

  if c.conn, err = dcerpc.Dial(ctx, c.Host, do...); err != nil {

    log.Error().Err(err).Msgf("Failed to connect to %s endpoint", c.String())
    return fmt.Errorf("dial %s: %w", c.String(), err)
  }

  return
}

func (c *Client) Close(ctx context.Context) (err error) {
  return c.conn.Close(ctx)
}
