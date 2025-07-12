package dcomexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/oaut"
  "github.com/rs/zerolog"
)

const (
  MethodShellWindows = "ShellWindows" // ShellWindows::Item().Document.Application.ShellExecute
)

type DcomShellWindows struct {
  Dcom

  IO goexec.ExecutionIO

  WorkingDirectory string
  WindowState      string
}

// Execute will perform command execution via the ShellWindows object. See https://enigma0x3.net/2017/01/23/lateral-movement-via-dcom-round-2/
func (m *DcomShellWindows) Execute(ctx context.Context, execIO *goexec.ExecutionIO) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", ModuleName).
    Str("method", MethodShellWindows).
    Logger()

  method := "Item"

  cmdline := execIO.CommandLine()
  proc := cmdline[0]
  args := cmdline[1]

  iv, err := callComMethod(ctx,
    m.dispatchClient,
    nil,
    "Item")

  if err != nil {
    log.Error().Err(err).Msg("Failed to call method")
    return fmt.Errorf("call method %q: %w", method, err)
  }

  item, ok := iv.VarResult.VarUnion.GetValue().(*oaut.Dispatch)
  if !ok {
    return errors.New("failed to get dispatch from ShellWindows::Item()")
  }

  method = "Document.Application.ShellExecute"

  // Arguments must be passed in reverse order
  if _, err := callComMethod(ctx, m.dispatchClient,
    item.InterfacePointer().
      GetStandardObjectReference().
      Std.IPID,
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
