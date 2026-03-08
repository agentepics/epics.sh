package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/fsutil"
	"github.com/agentepics/epics.sh/internal/testutil"
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
	root := testutil.RepoRoot(t)
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

func TestRunInstallHooksRejectsEmptyPromptHook(t *testing.T) {
	root := testutil.RepoRoot(t)
	src := filepath.Join(root, "examples", "fixtures", "invalid-prompt-install-hook-epic")
	dest := filepath.Join(t.TempDir(), "invalid-prompt-install-hook-epic")

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

	err = RunInstallHooks(pkg)
	if err == nil {
		t.Fatal("expected empty prompt hook error")
	}
	if !strings.Contains(err.Error(), "body is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallRollsBackOnHookFailure(t *testing.T) {
	root := testutil.RepoRoot(t)
	src := filepath.Join(root, "examples", "fixtures", "failing-install-hook-epic")
	cwd := t.TempDir()
	dest := filepath.Join(cwd, ".claude", "skills", "failing-install-hook-epic")

	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "existing.txt"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Install(cwd, src, "claude", func(slug string) string {
		return filepath.Join(cwd, ".claude", "skills", slug)
	})
	if err == nil {
		t.Fatal("expected install to fail")
	}
	if !strings.Contains(err.Error(), "install hook failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	// The previous install should remain intact because failed installs stay in staging.
	raw, readErr := os.ReadFile(filepath.Join(dest, "existing.txt"))
	if readErr != nil {
		t.Fatalf("expected previous install contents to remain: %v", readErr)
	}
	if string(raw) != "keep me\n" {
		t.Fatalf("unexpected previous install contents: %q", string(raw))
	}

	if _, statErr := os.Stat(filepath.Join(dest, "runtime", "failure-sentinel.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected failed staging output to be absent from destination, got: %v", statErr)
	}

	installs, loadErr := os.ReadFile(filepath.Join(cwd, ".epics", "installs.json"))
	if loadErr == nil {
		t.Fatalf("expected no install metadata to be written, got %s", string(installs))
	}
	if !os.IsNotExist(loadErr) {
		t.Fatalf("unexpected metadata error: %v", loadErr)
	}
}

func copyDir(src, dest string) error { return fsutil.CopyDir(src, dest) }
