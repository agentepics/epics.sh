package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInitCreatesPackage(t *testing.T) {
	dir := t.TempDir()
	app := NewApp(dir, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	if code := app.Run([]string{"init"}); code != 0 {
		t.Fatalf("expected success, got %d", code)
	}

	for _, name := range []string{"SKILL.md", "EPIC.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}

func TestValidateFixture(t *testing.T) {
	root := repoRoot(t)
	valid := filepath.Join(root, "examples", "fixtures", "valid-epic")
	invalid := filepath.Join(root, "examples", "fixtures", "invalid-missing-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(root, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"validate", valid}); code != 0 {
		t.Fatalf("expected valid fixture to pass, got %d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"validate", invalid}); code == 0 {
		t.Fatalf("expected invalid fixture to fail")
	}
}

func TestInstallLocalFixtureAndInfo(t *testing.T) {
	root := repoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"install", "--host", "claude", fixture}); code != 0 {
		t.Fatalf("install failed: code=%d stderr=%s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(workdir, ".claude", "skills", "resume-epic", "SKILL.md")); err != nil {
		t.Fatalf("expected installed package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workdir, ".claude", "commands", "epics-resume.md")); err != nil {
		t.Fatalf("expected Claude command file: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"info", "resume-epic"}); code != 0 {
		t.Fatalf("info failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Title: Resume Epic") {
		t.Fatalf("unexpected info output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Host: claude") {
		t.Fatalf("expected host in info output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Installed: .claude/skills/resume-epic") {
		t.Fatalf("expected installed path in info output: %s", stdout.String())
	}
}

func TestInstallRegistryEntry(t *testing.T) {
	root := repoRoot(t)
	workdir := t.TempDir()

	if err := copyDir(filepath.Join(root, "registry"), filepath.Join(workdir, "registry")); err != nil {
		t.Fatalf("copy registry: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"install", "--host", "claude", "github.com/agentepics/epics/autonomous-coding"}); code != 0 {
		t.Fatalf("registry install failed: code=%d stderr=%s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(workdir, ".claude", "skills", "autonomous-coding", "EPIC.md")); err != nil {
		t.Fatalf("expected installed registry package: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"info", "autonomous-coding"}); code != 0 {
		t.Fatalf("registry info failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Source: github.com/agentepics/epics/autonomous-coding") {
		t.Fatalf("expected source in info output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Host: claude") {
		t.Fatalf("expected host in info output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Installed: .claude/skills/autonomous-coding") {
		t.Fatalf("expected installed path in info output: %s", stdout.String())
	}
}

func TestInstallPromptsForHostWhenInteractive(t *testing.T) {
	root := repoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader("claude\n"), &stdout, &stderr)
	app.IsInteractive = func() bool { return true }

	if code := app.Run([]string{"install", fixture}); code != 0 {
		t.Fatalf("interactive install failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Select host:") {
		t.Fatalf("expected host prompt in stdout: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(workdir, ".claude", "skills", "resume-epic", "EPIC.md")); err != nil {
		t.Fatalf("expected interactive install package: %v", err)
	}
}

func TestInstallRequiresHostWhenNonInteractive(t *testing.T) {
	root := repoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)
	app.IsInteractive = func() bool { return false }

	if code := app.Run([]string{"install", fixture}); code == 0 {
		t.Fatalf("expected missing-host install to fail")
	}
	if !strings.Contains(stderr.String(), "install requires --host <host>") {
		t.Fatalf("expected missing-host error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestResumeUsesStateAndPlan(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(root, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"resume", fixture}); code != 0 {
		t.Fatalf("resume failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Next step: Verify the generated summary output") {
		t.Fatalf("expected next step in output, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Current plan: plans/001-current.md") {
		t.Fatalf("expected plan path in output, got %s", stdout.String())
	}
}

func TestDoctorJSON(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "doctor"}); code != 0 {
		t.Fatalf("doctor failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"checks\"") {
		t.Fatalf("unexpected doctor json output: %s", stdout.String())
	}
}

func TestHostSetupClaudeIsAdditive(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(existing, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("write existing claude: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"host", "setup", "claude"}); code != 0 {
		t.Fatalf("host setup failed: code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "commands", "epics-resume.md")); err != nil {
		t.Fatalf("expected Claude command file: %v", err)
	}
	raw, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read existing claude: %v", err)
	}
	if string(raw) != "# Existing\n" {
		t.Fatalf("expected existing CLAUDE.md to remain unchanged, got %q", string(raw))
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

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, info.Mode())
	})
}
