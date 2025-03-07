package wmiexec

import (
  "github.com/RedTeamPentesting/adauth"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemservices/v0"
  "github.com/rs/zerolog"
)

type Module struct {
  creds  *adauth.Credential
  target *adauth.Target

  log zerolog.Logger
  dce dcerpc.Conn
  sc  iwbemservices.ServicesClient
}

type MethodCustomConfig struct {
  Class     string
  Method    string
  Arguments map[string]any
}

type MethodProcessConfig struct {
  Command          string
  WorkingDirectory string
}

const (
  MethodCustom  = "custom"
  MethodProcess = "process"
)
