package dcomexec

import (
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"
)

type Module struct {
	dce      dcerpc.Conn
	dc       idispatch.DispatchClient
	hostname string
}

type MethodMmcConfig struct {
	WorkingDirectory string
	WindowState      string
}

const (
	MethodMmc string = "mmc"
)
