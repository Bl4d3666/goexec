package wmiexec

import (
  "context"
  "errors"
  "fmt"
  "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmio/query"
)

func (mod *Module) query(ctx context.Context, class, method string, values map[string]any) (outValues map[string]any, err error) {
  outValues = make(map[string]any)
  if mod.sc == nil {
    err = errors.New("module has not been initialized")
    return
  }
  if out, err := query.NewBuilder(ctx, mod.sc, ComVersion).
    Spawn(class). // The class to instantiate (i.e. Win32_Process)
    Method(method). // The method to call (i.e. Create)
    Values(values). // The values to pass to method
    Exec().
    Object(); err == nil {
    return out.Values(), err
  }
  err = fmt.Errorf("(*query.Builder).Spawn: %w", err)
  return
}
