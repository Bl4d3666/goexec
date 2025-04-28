package dcomexec

import (
	"context"
	"fmt"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"
	"strings"

	_ "github.com/oiweiwei/go-msrpc/msrpc/erref/ntstatus"
	_ "github.com/oiweiwei/go-msrpc/msrpc/erref/win32"
)

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
