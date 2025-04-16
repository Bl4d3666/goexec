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
  DcomExec

  WorkingDirectory string
  WindowState      string
}

// Execute will perform command execution via the MMC20.Application DCOM object.
func (m *DcomMmc) Execute(ctx context.Context, in *goexec.ExecutionInput) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", ModuleName).
    Str("method", MethodMmc).
    Logger()

  method := "Document.ActiveView.ExecuteShellCommand"

  var args = in.Arguments
  if args == "" {
    args = " " // the process arguments can't be a blank string
  }

  // Arguments must be passed in reverse order
  if _, err := callComMethod(ctx,
    m.dispatchClient,
    method,
    stringToVariant(m.WindowState),
    stringToVariant(in.Arguments),
    stringToVariant(m.WorkingDirectory),
    stringToVariant(in.Executable)); err != nil {

    log.Error().Err(err).Msg("Failed to call method")
    return fmt.Errorf("call %q: %w", method, err)
  }
  log.Info().Msg("Method call successful")
  return
}
