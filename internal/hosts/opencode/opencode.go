package opencode

import (
	"path/filepath"

	"github.com/agentepics/epics.sh/internal/doctor"
	"github.com/agentepics/epics.sh/internal/hostapi"
	"github.com/agentepics/epics.sh/internal/hostutil"
)

const epicGuidanceMarker = "## Epics CLI Guidance"

type Adapter struct{}

func (Adapter) Name() string {
	return "opencode"
}

func (Adapter) InstallDir(cwd, slug string) string {
	return filepath.Join(cwd, ".opencode", "skills", slug)
}

func (Adapter) Setup(cwd string) (hostapi.Result, error) {
	files := map[string]string{
		filepath.Join(".opencode", "commands", "epics-resume.md"): resumeCommand(),
		filepath.Join(".opencode", "commands", "epics-info.md"):   infoCommand(),
		filepath.Join(".opencode", "commands", "epics-doctor.md"): doctorCommand(),
	}

	var result hostapi.Result
	for relative, content := range files {
		state, err := hostutil.WriteIfMissingOrSame(filepath.Join(cwd, relative), content)
		if err != nil {
			return hostapi.Result{}, err
		}
		hostutil.RecordWrite(&result, relative, state)
	}

	state, err := hostutil.AppendSection(filepath.Join(cwd, "AGENTS.md"), guidance())
	if err != nil {
		return hostapi.Result{}, err
	}
	hostutil.RecordWrite(&result, "AGENTS.md", state)

	return result, nil
}

func (Adapter) Doctor(cwd string) ([]doctor.Check, error) {
	return hostutil.HostDoctorChecks(cwd, "opencode", ".opencode", "AGENTS.md", epicGuidanceMarker)
}

func guidance() string {
	return epicGuidanceMarker + "\n\n" +
		"This workspace uses the `epics` CLI as the canonical Epic control surface.\n\n" +
		"Use these OpenCode command prompts when they are available:\n\n" +
		"- `epics-resume` to reconstruct the current workflow context\n" +
		"- `epics-info` to inspect the installed Epic package\n" +
		"- `epics-doctor` to diagnose workspace and adapter setup\n\n" +
		"Keep Epic semantics in the package files and use OpenCode-specific files only as wrappers around the CLI."
}

func resumeCommand() string {
	return "Run `epics resume` in the repository root. Use the returned context to continue the current Epic without re-deriving state from scratch."
}

func infoCommand() string {
	return "Run `epics info` in the repository root. Summarize the active Epic package, source, and installed metadata before proceeding."
}

func doctorCommand() string {
	return "Run `epics doctor` in the repository root. Report any failing or warning checks before continuing with adapter-sensitive work."
}
