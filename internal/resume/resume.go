package resume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentepics/epics.sh/internal/epic"
)

type Result struct {
	Package        epic.Package   `json:"package"`
	StatePath      string         `json:"statePath,omitempty"`
	PlanPath       string         `json:"planPath,omitempty"`
	LogPaths       []string       `json:"logPaths,omitempty"`
	NextStep       string         `json:"nextStep,omitempty"`
	PlanExcerpt    string         `json:"planExcerpt,omitempty"`
	Context        string         `json:"context"`
	State          map[string]any `json:"state,omitempty"`
	RecentLogNotes []string       `json:"recentLogNotes,omitempty"`
}

func Build(pkg epic.Package) (Result, error) {
	result := Result{
		Package: pkg,
	}

	state, statePath, err := epic.ReadState(pkg)
	if err != nil {
		return Result{}, err
	}
	result.State = state
	result.StatePath = epic.RelativePath(pkg.Root, statePath)
	result.NextStep = epic.LookupString(state, "next", "nextStep", "now", "currentStep", "current_step")

	planPath := resolvePlanPath(pkg, state)
	if planPath != "" {
		raw, err := os.ReadFile(planPath)
		if err != nil {
			return Result{}, err
		}
		result.PlanPath = epic.RelativePath(pkg.Root, planPath)
		result.PlanExcerpt = epic.ExtractPlanExcerpt(string(raw))
	}

	logPaths := epic.LatestFiles(pkg.LogFiles, 3)
	result.LogPaths = make([]string, 0, len(logPaths))
	for _, logPath := range logPaths {
		raw, err := os.ReadFile(logPath)
		if err != nil {
			continue
		}
		result.LogPaths = append(result.LogPaths, epic.RelativePath(pkg.Root, logPath))
		note := epic.ExtractPlanExcerpt(string(raw))
		if note != "" {
			result.RecentLogNotes = append(result.RecentLogNotes, note)
		}
	}

	result.Context = renderContext(result)
	return result, nil
}

func resolvePlanPath(pkg epic.Package, state map[string]any) string {
	ref := epic.LookupString(
		state,
		"currentPlan",
		"current_plan",
		"currentPlanPath",
		"current_plan_path",
		"plan",
		"planPath",
	)
	if ref != "" {
		path := ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(pkg.Root, filepath.FromSlash(ref))
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	if len(pkg.PlanFiles) > 0 {
		return pkg.PlanFiles[len(pkg.PlanFiles)-1]
	}

	return ""
}

func renderContext(result Result) string {
	lines := []string{
		fmt.Sprintf("Epic: %s", result.Package.Title),
	}
	if result.Package.Summary != "" {
		lines = append(lines, "Summary: "+result.Package.Summary)
	}
	if result.StatePath != "" {
		lines = append(lines, "State: "+result.StatePath)
	}
	if result.PlanPath != "" {
		lines = append(lines, "Current plan: "+result.PlanPath)
	}
	if result.NextStep != "" {
		lines = append(lines, "Next step: "+result.NextStep)
	}
	if result.PlanExcerpt != "" {
		lines = append(lines, "Plan excerpt: "+result.PlanExcerpt)
	}
	if len(result.LogPaths) > 0 {
		lines = append(lines, "Recent logs: "+strings.Join(result.LogPaths, ", "))
	}
	if len(result.RecentLogNotes) > 0 {
		lines = append(lines, "Recent notes: "+strings.Join(result.RecentLogNotes, " | "))
	}
	if result.PlanPath == "" && len(result.LogPaths) == 0 {
		lines = append(lines, "Resume hint: review EPIC.md and SKILL.md to re-enter the workflow.")
	}
	return strings.Join(lines, "\n")
}
