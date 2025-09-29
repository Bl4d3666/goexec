package dcomexec

import (
  "context"
  "errors"
  "fmt"
  "strings"
  "syscall"

  "github.com/FalconOpsLLC/goexec/internal/util"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/oiweiwei/go-msrpc/midl/uuid"
  "github.com/oiweiwei/go-msrpc/msrpc/erref/hresult"
  "github.com/rs/zerolog"
)

const (
  MethodExcelXlm       = "Excel:ExecuteExcel4Macro"
  ExcelApplicationUuid = "00020812-0000-0000-C000-000000000046"
)

type DcomExcelXlm struct {
  Dispatch
  Macro     string
  MacroFile string
}

// Init will initialize the ShellBrowserWindow instance
func (m *DcomExcelXlm) Init(ctx context.Context) (err error) {
  if err = m.Dcom.Init(ctx); err == nil {
    return m.getDispatch(ctx, uuid.MustParse(ExcelApplicationUuid))
  }
  return
}

func (m *DcomExcelXlm) Execute(ctx context.Context, execIO *goexec.ExecutionIO) (err error) {
  log := zerolog.Ctx(ctx)
  if m.Macro == "" {
    m.Macro = fmt.Sprintf(`EXEC("%s")`, strings.ReplaceAll(execIO.String(), `"`, `""`))
  }
  { // Call ExecuteExcel4Macro to execute macro
    log.Info().
      Str("call", "ExecuteExcel4Macro").
      Str("macro", util.Truncate(m.Macro, 255)).
      Msg("executing Excel macro")
    ir, err := m.callComMethod(ctx, nil, "ExecuteExcel4Macro", stringToVariant(m.Macro))
    if err != nil {
      return err
    }
    if ir.Return != 0 {
      return hresult.FromCode(uint32(ir.Return))
    }
    log.Info().Msg("ExecuteExcel4Macro call successful")
  }
  { // Terminate EXCEL.EXE via ExecuteExcel4Macro("QUIT()")
    quit := "QUIT()"
    log.Info().
      Str("call", "ExecuteExcel4Macro").
      Str("macro", quit).
      Msg("terminating Excel process")
    qr, err := m.callComMethod(ctx, nil, "ExecuteExcel4Macro", stringToVariant(quit))
    _ = qr
    if err != nil {
      if errors.Is(err, syscall.ECONNRESET) {
        log.Info().Msg("Excel process terminated")
        return nil
      }
      return fmt.Errorf(`call ExecuteExcel4Macro("%s"): %w`, quit, err)
    }
    if qr.Return != 0 {
      return hresult.FromCode(uint32(qr.Return))
    }
  }
  return
}
