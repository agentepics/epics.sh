package hostapi

import "github.com/agentepics/epics.sh/internal/doctor"

type Result struct {
	Created   []string `json:"created,omitempty"`
	Unchanged []string `json:"unchanged,omitempty"`
	Skipped   []string `json:"skipped,omitempty"`
}

type Adapter interface {
	Name() string
	InstallDir(cwd, slug string) string
	Setup(cwd string) (Result, error)
	Doctor(cwd string) ([]doctor.Check, error)
}
