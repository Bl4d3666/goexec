package tschexec

import (
  "context"
  "encoding/xml"
  "fmt"
  "github.com/oiweiwei/go-msrpc/msrpc/tsch/itaskschedulerservice/v1"
  "github.com/rs/zerolog"
)

const (
  TaskXMLDurationFormat = "2006-01-02T15:04:05.9999999Z"
  TaskXMLHeader         = `<?xml version="1.0" encoding="UTF-16"?>`
)

// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/0d6383e4-de92-43e7-b0bb-a60cfa36379f

type triggers struct {
  XMLName      xml.Name          `xml:"Triggers"`
  TimeTriggers []taskTimeTrigger `xml:"TimeTrigger,omitempty"`
}

type taskTimeTrigger struct {
  XMLName       xml.Name `xml:"TimeTrigger"`
  StartBoundary string   `xml:"StartBoundary,omitempty"` // Derived from time.Time
  EndBoundary   string   `xml:"EndBoundary,omitempty"`   // Derived from time.Time; must be > StartBoundary
  Enabled       bool     `xml:"Enabled"`
}

type idleSettings struct {
  XMLName       xml.Name `xml:"IdleSettings"`
  StopOnIdleEnd bool     `xml:"StopOnIdleEnd"`
  RestartOnIdle bool     `xml:"RestartOnIdle"`
}

type settings struct {
  XMLName                    xml.Name     `xml:"Settings"`
  Enabled                    bool         `xml:"Enabled"`
  Hidden                     bool         `xml:"Hidden"`
  DisallowStartIfOnBatteries bool         `xml:"DisallowStartIfOnBatteries"`
  StopIfGoingOnBatteries     bool         `xml:"StopIfGoingOnBatteries"`
  AllowHardTerminate         bool         `xml:"AllowHardTerminate"`
  RunOnlyIfNetworkAvailable  bool         `xml:"RunOnlyIfNetworkAvailable"`
  AllowStartOnDemand         bool         `xml:"AllowStartOnDemand"`
  WakeToRun                  bool         `xml:"WakeToRun"`
  RunOnlyIfIdle              bool         `xml:"RunOnlyIfIdle"`
  StartWhenAvailable         bool         `xml:"StartWhenAvailable"`
  Priority                   int          `xml:"Priority,omitempty"` // 1 to 10 inclusive
  MultipleInstancesPolicy    string       `xml:"MultipleInstancesPolicy,omitempty"`
  ExecutionTimeLimit         string       `xml:"ExecutionTimeLimit,omitempty"`
  DeleteExpiredTaskAfter     string       `xml:"DeleteExpiredTaskAfter,omitempty"` // Derived from time.Duration
  IdleSettings               idleSettings `xml:"IdleSettings,omitempty"`
}

type actionExec struct {
  XMLName   xml.Name `xml:"Exec"`
  Command   string   `xml:"Command"`
  Arguments string   `xml:"Arguments,omitempty"`
}

type actions struct {
  XMLName xml.Name     `xml:"Actions"`
  Context string       `xml:"Context,attr"`
  Exec    []actionExec `xml:"Exec,omitempty"`
}

type principals struct {
  XMLName    xml.Name    `xml:"Principals"`
  Principals []principal `xml:"Principal,omitempty"`
}

type principal struct {
  XMLName  xml.Name `xml:"Principal"`
  ID       string   `xml:"id,attr"`
  UserID   string   `xml:"UserId"`
  RunLevel string   `xml:"RunLevel"`
}

type task struct {
  XMLName       xml.Name   `xml:"Task"`
  TaskVersion   string     `xml:"version,attr"`
  TaskNamespace string     `xml:"xmlns,attr"`
  Triggers      triggers   `xml:"Triggers"`
  Actions       actions    `xml:"Actions"`
  Principals    principals `xml:"Principals"`
  Settings      settings   `xml:"Settings"`
}

// registerTask serializes and submits the provided task structure
func (mod *Module) registerTask(ctx context.Context, taskDef task, taskPath string) (path string, err error) {

  var taskXml string

  log := zerolog.Ctx(ctx).With().
    Str("module", "tsch").
    Str("func", "createTask").Logger()

  // Generate task XML content. See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/0d6383e4-de92-43e7-b0bb-a60cfa36379f
  {
    doc, err := xml.Marshal(taskDef)
    if err != nil {
      log.Error().Err(err).Msg("failed to marshal task XML")
      return "", fmt.Errorf("marshal task: %w", err)
    }
    taskXml = TaskXMLHeader + string(doc)
    log.Debug().Str("content", taskXml).Msg("Generated task XML")
  }
  // Submit task
  {
    response, err := mod.tsch.RegisterTask(ctx, &itaskschedulerservice.RegisterTaskRequest{
      Path:       taskPath,
      XML:        taskXml,
      Flags:      0, // TODO
      LogonType:  0, // TASK_LOGON_NONE
      CredsCount: 0,
      Creds:      nil,
    })
    if err != nil {
      log.Error().Err(err).Msg("Failed to register task")
      return "", fmt.Errorf("register task: %w", err)
    }
    log.Info().Str("path", taskPath).Msg("Task created successfully")
    path = response.ActualPath
  }
  return
}

func (mod *Module) deleteTask(ctx context.Context, taskPath string) (err error) {

  log := zerolog.Ctx(ctx).With().
    Str("module", "tsch").
    Str("path", taskPath).
    Str("func", "deleteTask").Logger()

  if _, err = mod.tsch.Delete(ctx, &itaskschedulerservice.DeleteRequest{Path: taskPath}); err != nil {
    log.Error().Err(err).Msg("Failed to delete task")
    return fmt.Errorf("delete task: %w", err)
  }
  log.Info().Msg("Task deleted successfully")
  return
}
