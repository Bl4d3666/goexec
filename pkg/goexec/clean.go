package goexec

import (
  "context"
  "github.com/rs/zerolog"
)

type CleanProvider interface {
  Clean(ctx context.Context) (err error)
}

type Cleaner struct {
  workers []func(ctx context.Context) error
}

func (c *Cleaner) AddCleaner(worker func(ctx context.Context) error) {
  c.workers = append(c.workers, worker)
}

func (c *Cleaner) Clean(ctx context.Context) (err error) {
  log := zerolog.Ctx(ctx).With().
    Str("component", "cleaner").Logger()

  for _, worker := range c.workers {
    if err = worker(log.WithContext(ctx)); err != nil {

      log.Warn().Err(err).Msg("Clean worker failed")
    }
  }
  return
}
