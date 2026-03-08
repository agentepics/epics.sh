package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/agentepics/epics.sh/internal/epic"
)

func TestParseGitHubSource(t *testing.T) {
	t.Run("repo path", func(t *testing.T) {
		source, ok := ParseGitHubSource("github.com/agentepics/epics/autonomous-coding")
		if !ok {
			t.Fatal("expected repo-path source to parse")
		}
		if source.RepoURL != "https://github.com/agentepics/epics.git" {
			t.Fatalf("unexpected repo url: %s", source.RepoURL)
		}
		if source.Subpath != "autonomous-coding" {
			t.Fatalf("unexpected subpath: %s", source.Subpath)
		}
	})

	t.Run("tree url", func(t *testing.T) {
		source, ok := ParseGitHubSource("https://github.com/agentepics/epics/tree/main/autonomous-coding")
		if !ok {
			t.Fatal("expected tree url to parse")
		}
		if source.Branch != "main" {
			t.Fatalf("unexpected branch: %s", source.Branch)
		}
		if source.Subpath != "autonomous-coding" {
			t.Fatalf("unexpected subpath: %s", source.Subpath)
		}
	})
}

func TestRunInstallHooksExecutesShellHook(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "examples", "fixtures", "install-hook-epic")
	dest := filepath.Join(t.TempDir(), "install-hook-epic")

	if err := copyDir(src, dest); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	pkg, diagnostics, err := epic.Validate(dest)
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	if epic.HasErrors(diagnostics) {
		t.Fatalf("expected valid fixture, got diagnostics: %#v", diagnostics)
	}

	if err := RunInstallHooks(pkg); err != nil {
		t.Fatalf("run install hooks: %v", err)
	}

	recordPath := filepath.Join(dest, "runtime", "install.json")
	payloadPath := filepath.Join(dest, "runtime", "install-hook-output.json")
	for _, path := range []string{recordPath, payloadPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	recordRaw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read install record: %v", err)
	}
	if !strings.Contains(string(recordRaw), `"trigger": "install"`) {
		t.Fatalf("expected install record trigger, got %s", string(recordRaw))
	}

	payloadRaw, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("read hook payload: %v", err)
	}
	if !strings.Contains(string(payloadRaw), `"trigger":"install"`) {
		t.Fatalf("expected hook payload trigger, got %s", string(payloadRaw))
	}
}

func TestRunInstallHooksRejectsUnsupportedHTTPHook(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "EPIC.md"), []byte("---\nid: http-hook\n---\n\n# HTTP Hook\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks", "install.yaml"), []byte("type: http\nurl: https://example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, diagnostics, err := epic.Validate(dir)
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	if epic.HasErrors(diagnostics) {
		t.Fatalf("expected valid fixture, got diagnostics: %#v", diagnostics)
	}

	err = RunInstallHooks(pkg)
	if err == nil {
		t.Fatal("expected unsupported http hook error")
	}
	if !strings.Contains(err.Error(), "unsupported install hook type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
