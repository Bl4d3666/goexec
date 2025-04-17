package goexec

import (
  "bytes"
  "context"
  "fmt"
  "io"
  "os"
)

type OutputProvider interface {
  GetOutput(ctx context.Context, writer io.Writer) (err error)
  Clean(ctx context.Context) (err error)
}

type ExecutionIO struct {
  Cleaner

  Input  *ExecutionInput
  Output *ExecutionOutput
}

type ExecutionOutput struct {
  NoDelete   bool
  RemotePath string
  Provider   OutputProvider
  Writer     io.WriteCloser
}

type ExecutionInput struct {
  FilePath       string
  Executable     string
  ExecutablePath string
  Arguments      string
  CommandLine    string
}

func (execIO *ExecutionIO) GetOutput(ctx context.Context) (err error) {
  if execIO.Output.Provider != nil {
    return execIO.Output.Provider.GetOutput(ctx, execIO.Output.Writer)
  }
  return nil
}

func (execIO *ExecutionIO) CommandLine() string {
  return execIO.Input.Command()
}

func (execIO *ExecutionIO) Clean(ctx context.Context) (err error) {
  if execIO.Output.Provider != nil {
    return execIO.Output.Provider.Clean(ctx)
  }
  return nil
}

func (execIO *ExecutionIO) String() (cmd string) {

  cmd = execIO.Input.Command()

  if execIO.Output.Provider != nil && execIO.Output.RemotePath != "" {
    return fmt.Sprintf(`C:\Windows\System32\cmd.exe /C %s > %s`, cmd, execIO.Output.RemotePath)
  }
  return
}

func (i *ExecutionInput) Command() string {

  if i.CommandLine == "" {

    if i.ExecutablePath != "" {
      i.CommandLine = i.ExecutablePath

    } else if i.Executable != "" {
      i.CommandLine = i.Executable
    }

    if i.Arguments != "" {
      i.CommandLine += " " + i.Arguments
    }
  }
  return i.CommandLine
}

func (i *ExecutionInput) String() string {
  return i.Command()
}

func (i *ExecutionInput) UploadReader(_ context.Context) (reader io.Reader, err error) {

  if i.FilePath != "" {
    return os.OpenFile(i.FilePath, os.O_RDONLY, 0)
  }
  return bytes.NewBufferString(i.Command()), nil
}
