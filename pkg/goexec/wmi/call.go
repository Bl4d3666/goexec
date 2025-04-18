package wmiexec

import (
  "context"
  "encoding/json"
  "github.com/rs/zerolog"
)

type WmiCall struct {
  Wmi

  Class  string
  Method string
  Args   map[string]any
}

func (m *WmiCall) Call(ctx context.Context) (out []byte, err error) {
  var outMap map[string]any

  if outMap, err = m.query(ctx, m.Class, m.Method, m.Args); err != nil {
    return
  }
  zerolog.Ctx(ctx).Info().Msg("Call succeeded")

  out, err = json.Marshal(outMap)
  return
}
