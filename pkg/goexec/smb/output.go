package smb

import (
  "context"
  "errors"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "io"
  "os"
  "regexp"
  "time"
)

var (
  DefaultOutputPollInterval = 1 * time.Second
  DefaultOutputPollTimeout  = 60 * time.Second
  pathPrefix                = regexp.MustCompile(`^([a-zA-Z]:)?[\\/]*`)
)

type OutputFileFetcher struct {
  goexec.Cleaner

  Client *Client

  Share            string
  File             string
  DeleteOutputFile bool
  PollInterval     time.Duration
  PollTimeout      time.Duration

  relativePath string
}

func (o *OutputFileFetcher) GetOutput(ctx context.Context, writer io.Writer) (err error) {

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
  defer o.AddCleaners(o.Client.Close)

  err = o.Client.Mount(ctx, o.Share)
  if err != nil {
    return
  }

  stopAt := time.Now().Add(o.PollTimeout)
  var reader io.ReadCloser

  for {
    if time.Now().After(stopAt) {
      return errors.New("output timeout")
    }
    if reader, err = o.Client.mount.OpenFile(o.relativePath, os.O_RDONLY, 0); err == nil {
      break
    }
    time.Sleep(o.PollInterval)
  }

  if _, err = io.Copy(writer, reader); err != nil {
    return
  }

  o.AddCleaners(func(_ context.Context) error { return reader.Close() })

  if o.DeleteOutputFile {
    o.AddCleaners(func(_ context.Context) error {
      return o.Client.mount.Remove(o.relativePath)
    })
  }

  return
}
