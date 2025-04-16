package smb

import (
  "context"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "io"
  "os"
  "regexp"
  "time"
)

var (
  DefaultOutputPollInterval = 1 * time.Second
  DefaultOutputPollTimeout  = 60 * time.Second
  pathPrefix                = regexp.MustCompile(`^([a-zA-Z]:)?\\*`)
)

type OutputFileFetcher struct {
  goexec.Cleaner

  Client       *Client
  Share        string
  File         string
  PollInterval time.Duration
  PollTimeout  time.Duration

  relativePath string
}

func (o *OutputFileFetcher) GetOutput(ctx context.Context) (reader io.ReadCloser, err error) {

  if o.PollInterval == 0 {
    o.PollInterval = DefaultOutputPollInterval
  }
  if o.PollTimeout == 0 {
    o.PollTimeout = DefaultOutputPollTimeout
  }

  o.relativePath = pathPrefix.ReplaceAllString(o.File, "")

  err = o.Client.Connect(ctx)
  if err != nil {
    return
  }

  err = o.Client.Mount(ctx, o.Share)
  if err != nil {
    return
  }

  stopAt := time.Now().Add(o.PollTimeout)

  for {
    if time.Now().After(stopAt) {
      return
    }
    if reader, err = o.Client.mount.OpenFile(o.relativePath, os.O_RDONLY, 0); err == nil {
      return
    }
    time.Sleep(o.PollInterval)
  }
  return
}
