package smb

import (
  "context"
  "errors"
  "github.com/FalconOpsLLC/goexec/pkg/goexec"
  "github.com/rs/zerolog"
  "io"
  "os"
  "path/filepath"
  "regexp"
  "strings"
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
  SharePath        string
  File             string
  DeleteOutputFile bool
  ForceReconnect   bool
  PollInterval     time.Duration
  PollTimeout      time.Duration

  relativePath string
}

func (o *OutputFileFetcher) GetOutput(ctx context.Context, writer io.Writer) (err error) {

  log := zerolog.Ctx(ctx)

  if o.PollInterval == 0 {
    o.PollInterval = DefaultOutputPollInterval
  }
  if o.PollTimeout == 0 {
    o.PollTimeout = DefaultOutputPollTimeout
  }

  shp := pathPrefix.ReplaceAllString(strings.ToLower(strings.ReplaceAll(o.SharePath, `\`, "/")), "")
  fp := pathPrefix.ReplaceAllString(strings.ToLower(strings.ReplaceAll(o.File, `\`, "/")), "")

  if o.relativePath, err = filepath.Rel(shp, fp); err != nil {
    return
  }

  log.Info().Str("path", o.relativePath).Msg("Fetching output file")

  if o.ForceReconnect || !o.Client.connected {
    err = o.Client.Connect(ctx)
    if err != nil {
      return
    }
    defer o.AddCleaners(o.Client.Close)
  }

  if o.ForceReconnect || o.Client.share != o.Share {
    err = o.Client.Mount(ctx, o.Share)
    if err != nil {
      return
    }
  }

  stopAt := time.Now().Add(o.PollTimeout)
  var reader io.ReadCloser

  for {
    if time.Now().After(stopAt) {
      return errors.New("execution output timeout")
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
