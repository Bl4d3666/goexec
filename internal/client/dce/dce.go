package dce

import "github.com/oiweiwei/go-msrpc/dcerpc"

var (
  NP           = "ncacn_np"
  TCP          = "ncacn_ip_tcp"
  HTTP         = "ncacn_http"
  DefaultPorts = map[string]uint16{
    NP:   445,
    TCP:  135,
    HTTP: 593,
  }
)

type ConnectionMethodDCEConfig struct {
  NoEpm    bool // NoEpm disables EPM
  EpmAuto  bool // EpmAuto will find any suitable endpoint, without any filter
  Endpoint *dcerpc.StringBinding
  Options  []dcerpc.Option
}
