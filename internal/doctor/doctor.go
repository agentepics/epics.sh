package doctor

import (
	"os"
	"path/filepath"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/workspace"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Result struct {
	Checks []Check `json:"checks"`
}

func Run(cwd string) (Result, error) {
	checks := []Check{
		checkManagedDir(cwd),
		checkWorkspaceWritable(cwd),
	}

	if pkg, diagnostics, err := epic.Validate(cwd); err == nil && (pkg.SkillPath != "" || pkg.EpicPath != "") {
		status := "ok"
		message := "current directory contains a valid Epic package"
		if epic.HasErrors(diagnostics) {
			status = "fail"
			message = "current directory contains an invalid Epic package"
		}
		checks = append(checks, Check{Name: "current-package", Status: status, Message: message})
	} else {
		checks = append(checks, Check{Name: "current-package", Status: "ok", Message: "no Epic package detected in the current directory"})
	}

	return Result{Checks: checks}, nil
}

func HasFailures(result Result) bool {
	for _, check := range result.Checks {
		if check.Status == "fail" {
			return true
		}
	}
	return false
}

func checkManagedDir(cwd string) Check {
	if err := workspace.EnsureManagedDir(cwd); err != nil {
		return Check{Name: "managed-dir", Status: "fail", Message: err.Error()}
	}

	tempFile := filepath.Join(workspace.ManagedDir(cwd), ".doctor.tmp")
	if err := os.WriteFile(tempFile, []byte("ok\n"), 0o644); err != nil {
		return Check{Name: "managed-dir", Status: "fail", Message: err.Error()}
	}
	_ = os.Remove(tempFile)
	return Check{Name: "managed-dir", Status: "ok", Message: "workspace-managed install directory is writable"}
}

func checkWorkspaceWritable(cwd string) Check {
	tempFile := filepath.Join(cwd, ".epics-doctor.tmp")
	if err := os.WriteFile(tempFile, []byte("ok\n"), 0o644); err != nil {
		return Check{Name: "workspace-write", Status: "fail", Message: err.Error()}
	}
	_ = os.Remove(tempFile)
	return Check{Name: "workspace-write", Status: "ok", Message: "current workspace is writable"}
}
