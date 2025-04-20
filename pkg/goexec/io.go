package goexec

import (
  "context"
  "fmt"
  "io"
  "strings"
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
  StageFile      io.ReadCloser
  Executable     string
  ExecutablePath string
  Arguments      string
  Command        string
}

func (execIO *ExecutionIO) GetOutput(ctx context.Context) (err error) {
  if execIO.Output.Provider != nil {
    return execIO.Output.Provider.GetOutput(ctx, execIO.Output.Writer)
  }
  return nil
}

func (execIO *ExecutionIO) Clean(ctx context.Context) (err error) {
  if execIO.Output.Provider != nil {
    return execIO.Output.Provider.Clean(ctx)
  }
  return nil
}

func (execIO *ExecutionIO) CommandLine() (cmd []string) {
  if execIO.Output.Provider != nil && execIO.Output.RemotePath != "" {
    return []string{
      `C:\Windows\System32\cmd.exe`,
      fmt.Sprintf(`/C %s > %s 2>&1`, execIO.Input.String(), execIO.Output.RemotePath),
    }
  }
  return execIO.Input.CommandLine()
}

func (execIO *ExecutionIO) String() string {
  cmd := execIO.CommandLine()

  // Ensure that executable paths are quoted
  if strings.Contains(cmd[0], " ") {
    return fmt.Sprintf(`%q %s`, cmd[0], strings.Join(cmd[1:], " "))
  }
  return strings.Join(cmd, " ")
}

func (i *ExecutionInput) CommandLine() (cmd []string) {
  cmd = make([]string, 2)
  cmd[1] = i.Arguments

  switch {
  case i.Command != "":
    return strings.SplitN(i.Command, " ", 2)

  case i.ExecutablePath != "":
    cmd[0] = i.ExecutablePath

  case i.Executable != "":
    cmd[0] = i.Executable
  }

  return cmd
}

func (i *ExecutionInput) String() string {
  return strings.Join(i.CommandLine(), " ")
}

func (i *ExecutionInput) Reader() (reader io.Reader) {
  if i.StageFile != nil {
    return i.StageFile
  }
  return strings.NewReader(i.String())
}
