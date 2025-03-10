package dcomexec

import (
  "context"
  "fmt"
  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"
  "strings"
)

const (
  LC_ENGLISH_US uint32 = 0x409
)

func callMethod(ctx context.Context, dc idispatch.DispatchClient, method string, args ...*oaut.Variant) (ir *idispatch.InvokeResponse, err error) {
  parts := strings.Split(method, ".")

  var id *dcom.IPID
  var gr *idispatch.GetIDsOfNamesResponse

  for i, obj := range parts {
    var opts []dcerpc.CallOption
    if id != nil {
      opts = append(opts, dcom.WithIPID(id))
    }
    gr, err = dc.GetIDsOfNames(ctx, &idispatch.GetIDsOfNamesRequest{
      This:     ORPCThis,
      IID:      &dcom.IID{},
      Names:    []string{obj + "\x00"},
      LocaleID: LC_ENGLISH_US,
    }, opts...)

    if err != nil {
      return nil, fmt.Errorf("get dispatch ID of name %q: %w", obj, err)
    }
    if len(gr.DispatchID) < 1 {
      return nil, fmt.Errorf("dispatch ID of name %q not found", obj)
    }
    irq := &idispatch.InvokeRequest{
      This:             ORPCThis,
      DispatchIDMember: gr.DispatchID[0],
      IID:              &dcom.IID{},
      LocaleID:         LC_ENGLISH_US,
    }
    if i >= len(parts)-1 {
      irq.Flags = 1
      irq.DispatchParams = &oaut.DispatchParams{ArgsCount: uint32(len(args)), Args: args}
      return dc.Invoke(ctx, irq, opts...)
    }
    irq.Flags = 2
    ir, err = dc.Invoke(ctx, irq, opts...)
    if err != nil {
      return nil, fmt.Errorf("get properties of object %q: %w", obj, err)
    }
    di, ok := ir.VarResult.VarUnion.GetValue().(*oaut.Dispatch)
    if !ok {
      return nil, fmt.Errorf("invalid dispatch object for %q", obj)
    }
    id = di.InterfacePointer().GetStandardObjectReference().Std.IPID
  }

  return
}
