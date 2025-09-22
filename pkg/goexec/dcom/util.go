package dcomexec

import (
  "context"
  "fmt"
  "strings"

  "github.com/oiweiwei/go-msrpc/dcerpc"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/iobjectexporter/v0"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"

  _ "github.com/oiweiwei/go-msrpc/msrpc/erref/ntstatus"
  _ "github.com/oiweiwei/go-msrpc/msrpc/erref/win32"
)

// getCOMVersion uses IObjectExporter.ServerAlive2() to determine the COM version of the server.
func getCOMVersion(ctx context.Context, cc dcerpc.Conn) (ver *dcom.COMVersion, err error) {
  oe, err := iobjectexporter.NewObjectExporterClient(ctx, cc)
  if err != nil {
    return nil, err
  }
  srv, err := oe.ServerAlive2(ctx, &iobjectexporter.ServerAlive2Request{})
  if err != nil {
    return nil, err
  }
  return srv.COMVersion, nil
}

// callComMethod calls a COM method on a remote object using the IDispatch interface.
//
// The method is specified as a dot-separated string, e.g. "ShellWindows.Item" to call the Item method on the ShellWindows object.
//
// The method arguments are passed as *oaut.Variant.
//
// The method returns an *idispatch.InvokeResponse, which contains the result of the method call.
//
// The method will automatically follow the IDispatch interface to get the object specified in the method name, e.g. "ShellWindows.Item" will
// automatically call "ShellWindows.Item.QueryInterface" to get the IDispatch interface of the object, then call "Item.Invoke" to call the method.
func callComMethod(ctx context.Context, dc idispatch.DispatchClient, id *dcom.IPID, method string, args ...*oaut.Variant) (ir *idispatch.InvokeResponse, err error) {
  parts := strings.Split(method, ".")

  for i, obj := range parts {
    var opts []dcerpc.CallOption

    if id != nil {
      opts = append(opts, dcom.WithIPID(id))
    }
    gr, err := dc.GetIDsOfNames(ctx, &idispatch.GetIDsOfNamesRequest{
      This:     ORPCThis,
      IID:      &dcom.IID{},
      LocaleID: LcEnglishUs,

      Names: []string{obj + "\x00"},
    }, opts...)

    if err != nil {
      return nil, fmt.Errorf("get dispatch ID of name %q: %w", obj, err)
    }

    if len(gr.DispatchID) < 1 {
      return nil, fmt.Errorf("dispatch ID of name %q not found", obj)
    }

    irq := &idispatch.InvokeRequest{
      This:     ORPCThis,
      IID:      &dcom.IID{},
      LocaleID: LcEnglishUs,

      DispatchIDMember: gr.DispatchID[0],
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

// stringToVariant converts a string to a *oaut.Variant.
func stringToVariant(s string) *oaut.Variant {
  return &oaut.Variant{
    Size: 5,
    VT:   8,
    VarUnion: &oaut.Variant_VarUnion{
      Value: &oaut.Variant_VarUnion_BSTR{
        BSTR: &oaut.String{
          Data: s,
        },
      },
    },
  }
}
