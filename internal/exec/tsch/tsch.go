package tschexec

import (
	"encoding/xml"
)

// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/0d6383e4-de92-43e7-b0bb-a60cfa36379f

type taskTimeTrigger struct {
	XMLName       xml.Name `xml:"TimeTrigger"`
	StartBoundary string   `xml:"StartBoundary,omitempty"` // Derived from time.Time
	EndBoundary   string   `xml:"EndBoundary,omitempty"`   // Derived from time.Time; must be > StartBoundary
	Enabled       bool     `xml:"Enabled"`
}

type idleSettings struct {
	StopOnIdleEnd bool `xml:"StopOnIdleEnd"`
	RestartOnIdle bool `xml:"RestartOnIdle"`
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
	Arguments string   `xml:"Arguments"`
}

type actions struct {
	XMLName xml.Name     `xml:"Actions"`
	Context string       `xml:"Context,attr"`
	Exec    []actionExec `xml:"Exec,omitempty"`
}

type principals struct {
	XMLName    xml.Name    `xml:"Principals"`
	Principals []principal `xml:"Principal"`
}

type principal struct {
	XMLName  xml.Name `xml:"Principal"`
	ID       string   `xml:"id,attr"`
	UserID   string   `xml:"UserId"`
	RunLevel string   `xml:"RunLevel"`
}

type task struct {
	XMLName       xml.Name          `xml:"Task"`
	TaskVersion   string            `xml:"version,attr"`
	TaskNamespace string            `xml:"xmlns,attr"`
	TimeTriggers  []taskTimeTrigger `xml:"Triggers>TimeTrigger,omitempty"`
	Actions       actions           `xml:"Actions"`
	Principals    principals        `xml:"Principals"`
	Settings      settings          `xml:"Settings"`
}

var (
	defaultSettings = settings{
		MultipleInstancesPolicy:    "IgnoreNew",
		DisallowStartIfOnBatteries: false,
		StopIfGoingOnBatteries:     false,
		AllowHardTerminate:         true,
		RunOnlyIfNetworkAvailable:  false,
		IdleSettings: idleSettings{
			StopOnIdleEnd: true,
			RestartOnIdle: false,
		},
		AllowStartOnDemand: true,
		Enabled:            true,
		Hidden:             true,
		RunOnlyIfIdle:      false,
		WakeToRun:          false,
		Priority:           7, // 7 is a pretty standard value for scheduled tasks
		StartWhenAvailable: true,
	}
	defaultPrincipals = principals{
		Principals: []principal{
			{
				ID:       "SYSTEM",
				UserID:   "S-1-5-18",
				RunLevel: "HighestAvailable",
			},
		},
	}
)
