package tschexec

import (
  "fmt"
  "regexp"
  "time"
)

var (
  TaskPathRegex = regexp.MustCompile(`^\\[^ :/\\][^:/]*$`)
  TaskNameRegex = regexp.MustCompile(`^[^ :/\\][^:/\\]*$`)
)

// newSettings just creates a settings instance with the necessary values + a few dynamic ones
func newSettings(terminate, onDemand, startWhenAvailable bool) *settings {
  return &settings{
    MultipleInstancesPolicy: "IgnoreNew",
    AllowHardTerminate:      terminate,
    IdleSettings: idleSettings{
      StopOnIdleEnd: true,
      RestartOnIdle: false,
    },
    AllowStartOnDemand: onDemand,
    Enabled:            true,
    Hidden:             true,
    Priority:           7, // a pretty standard value for scheduled tasks
    StartWhenAvailable: startWhenAvailable,
  }
}

// newTask creates a task with any static values filled
func newTask(se *settings, pr []principal, tr triggers, cmd, args string) *task {
  if se == nil {
    se = newSettings(true, true, false)
  }
  if pr == nil || len(pr) == 0 {
    pr = []principal{
      {
        ID:       "1",
        UserID:   "S-1-5-18",
        RunLevel: "HighestAvailable",
      },
    }
  }
  return &task{
    TaskVersion:   "1.2",
    TaskNamespace: "http://schemas.microsoft.com/windows/2004/02/mit/task",
    Triggers:      tr,
    Principals:    principals{Principals: pr},
    Settings:      *se,
    Actions: actions{
      Context: pr[0].ID,
      Exec: []actionExec{
        {
          Command:   cmd,
          Arguments: args,
        },
      },
    },
  }
}

// xmlDuration is a *very* simple implementation of xs:duration - only accepts +seconds
func xmlDuration(dur time.Duration) string {
  if s := int(dur.Seconds()); s >= 0 {
    return fmt.Sprintf(`PT%dS`, s)
  }
  return `PT0S`
}

// ValidateTaskName will validate the provided task name according to https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/fa8809c8-4f0f-4c6d-994a-6c10308757c1
func ValidateTaskName(taskName string) error {
  if !TaskNameRegex.MatchString(taskName) {
    return fmt.Errorf("invalid task name: %s", taskName)
  }
  return nil
}

// ValidateTaskPath will validate the provided task path according to https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/fa8809c8-4f0f-4c6d-994a-6c10308757c1
func ValidateTaskPath(taskPath string) error {
  if !TaskPathRegex.MatchString(taskPath) {
    return fmt.Errorf("invalid task path: %s", taskPath)
  }
  return nil
}
