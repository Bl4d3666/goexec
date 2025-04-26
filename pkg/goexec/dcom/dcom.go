package dcomexec

import (
	googleUUID "github.com/google/uuid"
	"github.com/oiweiwei/go-msrpc/midl/uuid"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom"
	"github.com/oiweiwei/go-msrpc/msrpc/dtyp"
)

const (
	LcEnglishUs uint32 = 0x409
)

var (
	//ShellWindowsUuid = uuid.MustParse("9BA05972-F6A8-11CF-A442-00A0C90A8F39")
	//Mmc20Uuid        = uuid.MustParse("49B2791A-B1AE-4C90-9B8E-E860BA07F889")

	RandCid      = dcom.CID(*dtyp.GUIDFromUUID(uuid.MustParse(googleUUID.NewString())))
	IDispatchIID = &dcom.IID{
		Data1: 0x20400,
		Data2: 0x0,
		Data3: 0x0,
		Data4: []byte{0xc0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x46},
	}
	ComVersion = &dcom.COMVersion{
		MajorVersion: 5,
		MinorVersion: 7,
	}
	ORPCThis = &dcom.ORPCThis{
		Version: ComVersion,
		CID:     &RandCid,
	}
)
