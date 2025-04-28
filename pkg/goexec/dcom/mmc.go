package dcomexec

import (
	"context"
	"fmt"
	"github.com/FalconOpsLLC/goexec/pkg/goexec"
	"github.com/rs/zerolog"
)

const (
	MethodMmc = "MMC" // MMC20.Application::Document.ActiveView.ExecuteShellCommand
)

type DcomMmc struct {
	Dcom

	IO goexec.ExecutionIO

	WorkingDirectory string
	WindowState      string
}

// Execute will perform command execution via the MMC20.Application DCOM object.
func (m *DcomMmc) Execute(ctx context.Context, execIO *goexec.ExecutionIO) (err error) {

	log := zerolog.Ctx(ctx).With().
		Str("module", ModuleName).
		Str("method", MethodMmc).
		Logger()

	method := "Document.ActiveView.ExecuteShellCommand"

	cmdline := execIO.CommandLine()
	proc := cmdline[0]
	args := cmdline[1]

	// Arguments must be passed in reverse order
	if _, err := callComMethod(ctx,
		m.dispatchClient,
		nil,
		method,
		stringToVariant(m.WindowState),
		stringToVariant(args),
		stringToVariant(m.WorkingDirectory),
		stringToVariant(proc)); err != nil {

		log.Error().Err(err).Msg("Failed to call method")
		return fmt.Errorf("call %q: %w", method, err)
	}
	log.Info().Msg("Method call successful")
	return
}
