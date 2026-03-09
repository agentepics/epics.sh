package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentepics/epics.sh/internal/fsutil"
	"github.com/agentepics/epics.sh/internal/testutil"
)

func TestRunEmptyWorkspace(t *testing.T) {
	dir := t.TempDir()

	result, err := Run(dir)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	checks := checksByName(result)
	if checks["authored-package"].Status != "ok" {
		t.Fatalf("unexpected authored-package check: %+v", checks["authored-package"])
	}
	if !strings.Contains(checks["authored-package"].Message, "no authored Epic package") {
		t.Fatalf("unexpected authored-package message: %s", checks["authored-package"].Message)
	}
	if checks["installed-epics"].Status != "ok" {
		t.Fatalf("unexpected installed-epics check: %+v", checks["installed-epics"])
	}
	if checks["install-sources"].Status != "ok" {
		t.Fatalf("unexpected install-sources check: %+v", checks["install-sources"])
	}
}

func TestRunInstalledEpicNoAuthoredPackage(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	source := filepath.Join(root, "examples", "fixtures", "resume-epic")
	installed := filepath.Join(workdir, ".claude", "skills", "resume-epic")

	if err := fsutil.CopyDir(source, installed); err != nil {
		t.Fatalf("copy installed epic: %v", err)
	}
	writeInstalls(t, workdir, []map[string]any{{
		"slug":         "resume-epic",
		"title":        "Resume Epic",
		"host":         "claude",
		"source":       "./fixtures/resume-epic",
		"installedAt":  "2026-03-08T00:00:00Z",
		"installedDir": ".claude/skills/resume-epic",
	}})
	if err := fsutil.CopyDir(source, filepath.Join(workdir, "fixtures", "resume-epic")); err != nil {
		t.Fatalf("copy local source: %v", err)
	}

	result, err := Run(workdir)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	checks := checksByName(result)
	if !strings.Contains(checks["authored-package"].Message, "workspace tracks 1 installed Epic") {
		t.Fatalf("unexpected authored-package message: %s", checks["authored-package"].Message)
	}
	if checks["installed-epics"].Status != "ok" {
		t.Fatalf("unexpected installed-epics check: %+v", checks["installed-epics"])
	}
	if checks["install-sources"].Status != "ok" {
		t.Fatalf("unexpected install-sources check: %+v", checks["install-sources"])
	}
}

func TestRunMissingLocalSourceWarning(t *testing.T) {
	workdir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workdir, ".claude", "skills", "resume-epic"), 0o755); err != nil {
		t.Fatalf("mkdir installed dir: %v", err)
	}
	writeInstalls(t, workdir, []map[string]any{{
		"slug":         "resume-epic",
		"title":        "Resume Epic",
		"host":         "claude",
		"source":       "./fixtures/resume-epic",
		"installedAt":  "2026-03-08T00:00:00Z",
		"installedDir": ".claude/skills/resume-epic",
	}})

	result, err := Run(workdir)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	check := checksByName(result)["install-sources"]
	if check.Status != "warning" {
		t.Fatalf("expected warning, got %+v", check)
	}
	if !strings.Contains(check.Message, "missing sources:") {
		t.Fatalf("unexpected install-sources message: %s", check.Message)
	}
}

func TestRunSourceDriftWarning(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	source := filepath.Join(root, "examples", "fixtures", "resume-epic")
	installed := filepath.Join(workdir, ".claude", "skills", "resume-epic")
	localSource := filepath.Join(workdir, "fixtures", "resume-epic")

	if err := fsutil.CopyDir(source, installed); err != nil {
		t.Fatalf("copy installed epic: %v", err)
	}
	if err := fsutil.CopyDir(source, localSource); err != nil {
		t.Fatalf("copy local source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installed, "EPIC.md"), []byte("# Drifted\n"), 0o644); err != nil {
		t.Fatalf("write drifted epic: %v", err)
	}
	writeInstalls(t, workdir, []map[string]any{{
		"slug":         "resume-epic",
		"title":        "Resume Epic",
		"host":         "claude",
		"source":       "./fixtures/resume-epic",
		"installedAt":  "2026-03-08T00:00:00Z",
		"installedDir": ".claude/skills/resume-epic",
	}})

	result, err := Run(workdir)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}

	check := checksByName(result)["install-sources"]
	if check.Status != "warning" {
		t.Fatalf("expected warning, got %+v", check)
	}
	if !strings.Contains(check.Message, "installed copy differs from local source") {
		t.Fatalf("unexpected install-sources message: %s", check.Message)
	}
}

func checksByName(result Result) map[string]Check {
	checks := make(map[string]Check, len(result.Checks))
	for _, check := range result.Checks {
		checks[check.Name] = check
	}
	return checks
}

func writeInstalls(t *testing.T, cwd string, installs []map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(map[string]any{"installs": installs}, "", "  ")
	if err != nil {
		t.Fatalf("marshal installs: %v", err)
	}
	raw = append(raw, '\n')
	path := filepath.Join(cwd, ".epics", "installs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir installs dir: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write installs file: %v", err)
	}
}
