package dcomexec

import (
	"context"
	"fmt"
	"github.com/FalconOpsLLC/goexec/pkg/goexec"
	"github.com/rs/zerolog"
)

const (
	MethodShellBrowserWindow = "ShellBrowserWindow" // ShellBrowserWindow::Document.Application.ShellExecute
)

type DcomShellBrowserWindow struct {
	Dcom

	IO goexec.ExecutionIO

	WorkingDirectory string
	WindowState      string
}

// Execute will perform command execution via the ShellBrowserWindow object. See https://enigma0x3.net/2017/01/23/lateral-movement-via-dcom-round-2/
func (m *DcomShellBrowserWindow) Execute(ctx context.Context, execIO *goexec.ExecutionIO) (err error) {

	log := zerolog.Ctx(ctx).With().
		Str("module", ModuleName).
		Str("method", MethodShellBrowserWindow).
		Logger()

	method := "Document.Application.ShellExecute"

	cmdline := execIO.CommandLine()
	proc := cmdline[0]
	args := cmdline[1]

	// Arguments must be passed in reverse order
	if _, err := callComMethod(ctx, m.dispatchClient,
		nil,
		method,
		stringToVariant(m.WindowState),
		stringToVariant(""), // FUTURE?
		stringToVariant(m.WorkingDirectory),
		stringToVariant(args),
		stringToVariant(proc)); err != nil {

		log.Error().Err(err).Msg("Failed to call method")
		return fmt.Errorf("call %q: %w", method, err)
	}
	log.Info().Msg("Method call successful")
	return
}
