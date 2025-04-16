package goexec

import "context"

type ExecutionProvider interface {
  Execute(ctx context.Context, in *ExecutionInput) (err error)
}

type Executor struct{}

type CleanExecutionProvider interface {
  ExecutionProvider
  CleanProvider
}
