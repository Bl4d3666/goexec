package goexec

import (
  "context"
  "fmt"
  "github.com/rs/zerolog"
)

type Method interface{}

type RemoteMethod interface {
  Connect(ctx context.Context) error
  Init(ctx context.Context) error
}

type RemoteExecuteMethod interface {
  RemoteMethod
  Execute(ctx context.Context, io *ExecutionIO) error
}

type RemoteExecuteCleanMethod interface {
  RemoteExecuteMethod
  Clean(ctx context.Context) error
}

func ExecuteMethod(ctx context.Context, module RemoteExecuteMethod, execIO *ExecutionIO) (err error) {

  log := zerolog.Ctx(ctx)

  if err = module.Connect(ctx); err != nil {
    log.Error().Err(err).Msg("Connection failed")
    return fmt.Errorf("connect: %w", err)
  }

  if err = module.Init(ctx); err != nil {
    log.Error().Err(err).Msg("Module initialization failed")
    return fmt.Errorf("init module: %w", err)
  }

  if err = module.Execute(ctx, execIO); err != nil {
    log.Error().Err(err).Msg("Execution failed")
    return fmt.Errorf("execute: %w", err)
  }

  return
}

func ExecuteCleanMethod(ctx context.Context, module RemoteExecuteCleanMethod, execIO *ExecutionIO) (err error) {

  log := zerolog.Ctx(ctx)

  defer func() {
    if err = module.Clean(ctx); err != nil {
      log.Error().Err(err).Msg("Module cleanup failed")
      err = nil
    }
  }()

  if err = ExecuteMethod(ctx, module, execIO); err != nil {
    return
  }

  if execIO.Output != nil && execIO.Output.Provider != nil {
    defer execIO.Output.Provider.Clean(ctx)

    execIO.Output.Provider.GetOutput(ctx, execIO.Output.Writer)
  }
  return
}
