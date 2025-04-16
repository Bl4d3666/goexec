package goexec

import "context"

type Module interface {
  Init(ctx context.Context) error
}

type ExecutionModule interface {
  Module
  ExecutionProvider
}

type CleanExecutionModule interface {
  Module
  ExecutionProvider
  CleanProvider
}

type CleanExecutionOutputModule interface {
  Module
  ExecutionProvider
  CleanProvider
  OutputProvider
}
