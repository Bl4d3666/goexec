package dcomexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/FalconOpsLLC/goexec/pkg/goexec"
	"github.com/FalconOpsLLC/goexec/pkg/goexec/dce"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/midl/uuid"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom/iremotescmactivator/v0"
	"github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut/idispatch/v0"
	"github.com/oiweiwei/go-msrpc/msrpc/dtyp"
	"github.com/rs/zerolog"

	_ "github.com/oiweiwei/go-msrpc/msrpc/erref/ntstatus"
	_ "github.com/oiweiwei/go-msrpc/msrpc/erref/win32"
)

const (
	ModuleName = "DCOM"
)

type Dcom struct {
	goexec.Cleaner
	goexec.Executor

	Client  *dce.Client
	ClassID string

	dispatchClient idispatch.DispatchClient
}

func (m *Dcom) Connect(ctx context.Context) (err error) {

	if err = m.Client.Connect(ctx); err == nil {
		m.AddCleaners(m.Client.Close)
	}
	return
}

func (m *Dcom) Init(ctx context.Context) (err error) {

	log := zerolog.Ctx(ctx).With().
		Str("module", ModuleName).Logger()

	if m.Client == nil || m.Client.Dce() == nil {
		return errors.New("DCE connection not initialized")
	}

	m.ClassID = "49B2791A-B1AE-4C90-9B8E-E860BA07F889"
	//m.ClassID = "9BA05972-F6A8-11CF-A442-00A0C90A8F39"
	class := dcom.ClassID(*dtyp.GUIDFromUUID(uuid.MustParse(m.ClassID)))

	if class.GUID() == nil {
		return fmt.Errorf("invalid class ID: %s", m.ClassID)
	}

	opts := []dcerpc.Option{
		dcerpc.WithSign(),
	}

	inst := &dcom.InstantiationInfoData{
		ClassID:          &class,
		IID:              []*dcom.IID{IDispatchIID},
		ClientCOMVersion: ComVersion,
	}
	ac := &dcom.ActivationContextInfoData{}
	loc := &dcom.LocationInfoData{}
	scm := &dcom.SCMRequestInfoData{
		RemoteRequest: &dcom.CustomRemoteRequestSCMInfo{
			RequestedProtocolSequences: []uint16{7},
		},
	}

	ap := &dcom.ActivationProperties{
		DestinationContext: 2,
		Properties:         []dcom.ActivationProperty{inst, ac, loc, scm},
	}

	apin, err := ap.ActivationPropertiesIn()
	if err != nil {
		return err
	}

	act, err := iremotescmactivator.NewRemoteSCMActivatorClient(ctx, m.Client.Dce())
	if err != nil {
		return err
	}

	cr, err := act.RemoteCreateInstance(ctx, &iremotescmactivator.RemoteCreateInstanceRequest{
		ORPCThis: &dcom.ORPCThis{
			Version: ComVersion,
			Flags:   1,
			CID:     &RandCid,
		},
		ActPropertiesIn: apin,
	})
	if err != nil {
		return err
	}
	log.Info().Msg("RemoteCreateInstance succeeded")

	apout := new(dcom.ActivationProperties)
	if err = apout.Parse(cr.ActPropertiesOut); err != nil {
		return err
	}
	si := apout.SCMReplyInfoData()
	pi := apout.PropertiesOutInfo()

	if si == nil {
		return fmt.Errorf("remote create instance response: SCMReplyInfoData is nil")
	}

	if pi == nil {
		return fmt.Errorf("remote create instance response: PropertiesOutInfo is nil")
	}

	// Ensure that the string bindings don't contain the target hostname
	for _, bind := range si.RemoteReply.OXIDBindings.GetStringBindings() {
		stringBinding, err := dcerpc.ParseStringBinding("ncacn_ip_tcp:" + bind.NetworkAddr) // TODO: try bind.String()

		if err != nil {
			log.Debug().Err(err).Msg("Failed to parse string binding")
			continue
		}
		stringBinding.NetworkAddress = ""
		opts = append(opts, dcerpc.WithEndpoint(stringBinding.String()))
	}

	err = m.Client.Reconnect(ctx, opts...)
	if err != nil {
		return err
	}
	log.Info().Msg("created new DCERPC dialer")

	m.dispatchClient, err = idispatch.NewDispatchClient(ctx, m.Client.Dce(), dcom.WithIPID(pi.InterfaceData[0].IPID()))
	if err != nil {
		return err
	}
	log.Info().Msg("created IDispatch Client")

	return
}
