package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentepics/epics.sh/internal/daemon"
	daemonstore "github.com/agentepics/epics.sh/internal/daemon/store"
	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/fsutil"
	"github.com/agentepics/epics.sh/internal/testutil"
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

	skillRaw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	skill := string(skillRaw)
	if !strings.Contains(skill, "name: "+filepath.Base(dir)) {
		t.Fatalf("expected generated skill name, got %q", skill)
	}
	if !strings.Contains(skill, epic.CanonicalSkillFooterMarker) {
		t.Fatalf("expected canonical footer marker, got %q", skill)
	}

	epicRaw, err := os.ReadFile(filepath.Join(dir, "EPIC.md"))
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	if !strings.Contains(string(epicRaw), "spec_version: 0.5.2") {
		t.Fatalf("expected spec_version 0.5.2, got %q", string(epicRaw))
	}
}

func TestUpgradeSkillFooterCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test-epic\ndescription: Test fixture.\n---\n\n# Test Epic\n\nUse this epic when you need test coverage. `EPIC.md` is authoritative.\n\n## Agent Epics\n<!-- epics-canonical-footer: https://github.com/agentepics/agentepics/blob/v0.5.1/footer.md -->\n\nOld footer.\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"upgrade-skill-footer"}); code != 0 {
		t.Fatalf("upgrade command failed: code=%d stderr=%s", code, stderr.String())
	}

	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read updated skill: %v", err)
	}
	if !strings.Contains(string(raw), epic.CanonicalSkillFooterMarker) {
		t.Fatalf("expected canonical footer marker, got %q", string(raw))
	}
}

func TestValidateFixture(t *testing.T) {
	root := testutil.RepoRoot(t)
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
	root := testutil.RepoRoot(t)
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
	root := testutil.RepoRoot(t)
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
	root := testutil.RepoRoot(t)
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
	root := testutil.RepoRoot(t)
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
	root := testutil.RepoRoot(t)
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
	if !strings.Contains(stdout.String(), "Current plan: runtime/plans/001-current.md") {
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

func TestDoctorInstalledEpicDistinguishesAuthoredFromInstalled(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"install", "--host", "claude", fixture}); code != 0 {
		t.Fatalf("install failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"doctor"}); code != 0 {
		t.Fatalf("doctor failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK: authored-package - no authored Epic package in the current directory; workspace tracks 1 installed Epic(s)") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "OK: installed-epics - workspace metadata tracks 1 installed Epic(s): resume-epic@claude") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}

func TestDoctorWarnsWhenLocalSourceIsMissing(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	fixtureSource := filepath.Join(workdir, "fixtures", "resume-epic")
	if err := copyDir(filepath.Join(root, "examples", "fixtures", "resume-epic"), fixtureSource); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)
	if code := app.Run([]string{"install", "--host", "claude", "./fixtures/resume-epic"}); code != 0 {
		t.Fatalf("install failed: code=%d stderr=%s", code, stderr.String())
	}
	if err := os.RemoveAll(filepath.Join(workdir, "fixtures")); err != nil {
		t.Fatalf("remove fixture source: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"doctor"}); code != 0 {
		t.Fatalf("doctor failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "WARNING: install-sources - missing sources: resume-epic@claude ->") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}

func TestInfoRejectsExtraArgs(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"info", "one", "two"}); code == 0 {
		t.Fatal("expected info with extra args to fail")
	}
	if !strings.Contains(stderr.String(), "expected at most one argument") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
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
	content := string(raw)
	if !strings.Contains(content, "# Existing\n") {
		t.Fatalf("expected existing CLAUDE.md content preserved, got %q", content)
	}
	if !strings.Contains(content, "## Epics CLI Guidance") {
		t.Fatalf("expected Epic guidance appended, got %q", content)
	}
}

func TestHostDoctorJSON(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"install", "--host", "claude", fixture}); code != 0 {
		t.Fatalf("install failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"--json", "host", "doctor", "claude"}); code != 0 {
		t.Fatalf("host doctor failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"claude-managed-dir\"") {
		t.Fatalf("unexpected host doctor json output: %s", stdout.String())
	}
}

func TestStatusForInstalledEpic(t *testing.T) {
	root := testutil.RepoRoot(t)
	workdir := t.TempDir()
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workdir, strings.NewReader(""), &stdout, &stderr)
	if code := app.Run([]string{"install", "--host", "claude", fixture}); code != 0 {
		t.Fatalf("install failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"status", "resume-epic"}); code != 0 {
		t.Fatalf("status failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Epic: Resume Epic") {
		t.Fatalf("unexpected status output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Current plan: runtime/plans/001-current.md") {
		t.Fatalf("unexpected status output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Latest log: runtime/log/2026-03-08-01.md") {
		t.Fatalf("unexpected status output: %s", stdout.String())
	}
}

func TestStatusJSON(t *testing.T) {
	root := testutil.RepoRoot(t)
	fixture := filepath.Join(root, "examples", "fixtures", "resume-epic")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(root, strings.NewReader(""), &stdout, &stderr)
	if code := app.Run([]string{"--json", "status", fixture}); code != 0 {
		t.Fatalf("status failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"nextStep\": \"Verify the generated summary output\"") {
		t.Fatalf("unexpected status json output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"latestLogPath\": \"runtime/log/2026-03-08-01.md\"") {
		t.Fatalf("unexpected status json output: %s", stdout.String())
	}
}

func TestStateGetJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{\"phase\":{\"current\":\"planning\"}}\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "state", "get", "phase.current"}); code != 0 {
		t.Fatalf("state get failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"key\": \"phase.current\"") {
		t.Fatalf("expected key in output, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"planning\"") {
		t.Fatalf("expected value in output, got %s", stdout.String())
	}
}

func TestStateSetAndGet(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"state", "set", "phase.current", "\"planning\""}); code != 0 {
		t.Fatalf("state set failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"state", "get", "phase.current"}); code != 0 {
		t.Fatalf("state get failed: code=%d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "planning" {
		t.Fatalf("expected planning, got %q", stdout.String())
	}
}

func TestPlanCurrentJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "plans"), 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plans", "001-current.md"), []byte("# Current\n\nPlan body\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "plan", "current"}); code != 0 {
		t.Fatalf("plan current failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"path\": \"plans/001-current.md\"") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Plan body") {
		t.Fatalf("expected plan content in output: %s", stdout.String())
	}
}

func TestPlanCreateAndList(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"plan", "create", "Initial", "plan"}); code != 0 {
		t.Fatalf("plan create failed: code=%d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "plans/001-initial-plan.md" {
		t.Fatalf("unexpected create output: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"--json", "plan", "list"}); code != 0 {
		t.Fatalf("plan list failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"path\": \"plans/001-initial-plan.md\"") {
		t.Fatalf("unexpected list output: %s", stdout.String())
	}
}

func TestLogRecentJSON(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "2026-03-08-first.md"), []byte("first\n"), 0o644); err != nil {
		t.Fatalf("write first log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "2026-03-09-second.md"), []byte("second\n"), 0o644); err != nil {
		t.Fatalf("write second log: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "log", "recent", "1"}); code != 0 {
		t.Fatalf("log recent failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"content\": \"") {
		t.Fatalf("unexpected log json output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "second") {
		t.Fatalf("expected most recent log content, got %s", stdout.String())
	}
}

func TestLogCreate(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"log", "create", "Session", "1"}); code != 0 {
		t.Fatalf("log create failed: code=%d stderr=%s", code, stderr.String())
	}
	path := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(path, "log/") || !strings.HasSuffix(path, "-session-1.md") {
		t.Fatalf("unexpected created path %q", path)
	}
}

func TestCronValidateJSON(t *testing.T) {
	dir := t.TempDir()
	cronDir := filepath.Join(dir, "cron.d")
	if err := os.MkdirAll(cronDir, 0o755); err != nil {
		t.Fatalf("mkdir cron: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cronDir, "nightly"), []byte("bad line\n"), 0o644); err != nil {
		t.Fatalf("write cron file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(dir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "cron", "validate"}); code != 1 {
		t.Fatalf("cron validate returned wrong code: code=%d stderr=%s", code, stderr.String())
	}

	var diagnostics []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &diagnostics); err != nil {
		t.Fatalf("unmarshal diagnostics: %v", err)
	}
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got %s", stdout.String())
	}
}

func TestDaemonWorkspaceAndRouteCommands(t *testing.T) {
	home := newShortCLIDir(t, "epicsd-cli-home")
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	startTestDaemonForCLI(t, home)
	t.Setenv("EPICSD_HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(workspaceDir, strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"workspace", "register", "--name", "repo-a"}); code != 0 {
		t.Fatalf("workspace register failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "repo-a") {
		t.Fatalf("unexpected workspace register output: %s", stdout.String())
	}
	workspaceID := strings.Fields(strings.TrimSpace(stdout.String()))[0]

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"route", "upsert", "--type", "webhook", "--workspace", workspaceID, "--epic", "resume-epic", "--provider", "github", "--endpoint", "repo-a", "--preferred-adapter", "claude", "--auth", "bearer", "--secret", "token"}); code != 0 {
		t.Fatalf("route upsert failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "webhook:github:repo-a") {
		t.Fatalf("unexpected route upsert output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"run", "list"}); code != 0 {
		t.Fatalf("run list failed: code=%d stderr=%s", code, stderr.String())
	}
}

func TestDaemonStatusJSON(t *testing.T) {
	home := newShortCLIDir(t, "epicsd-cli-home")
	startTestDaemonForCLI(t, home)
	t.Setenv("EPICSD_HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(t.TempDir(), strings.NewReader(""), &stdout, &stderr)

	if code := app.Run([]string{"--json", "daemon", "status"}); code != 0 {
		t.Fatalf("daemon status failed: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"webhookHTTPAddr\"") {
		t.Fatalf("unexpected daemon status output: %s", stdout.String())
	}
}

func startTestDaemonForCLI(t *testing.T, home string) {
	t.Helper()
	binDir := t.TempDir()
	epicsPath := filepath.Join(binDir, "epics")
	claudePath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(epicsPath, []byte("#!/bin/sh\necho \"resume:$2\"\n"), 0o755); err != nil {
		t.Fatalf("write epics stub: %v", err)
	}
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write claude stub: %v", err)
	}

	st := daemonstore.Open(home)
	cfg := daemonstore.DefaultConfig(home)
	cfg.WebhookHTTPAddr = "127.0.0.1:0"
	cfg.AdminSocketPath = filepath.Join(home, "epicsd.sock")
	if err := st.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	server, err := daemon.New(daemon.Options{
		Home:         home,
		EpicsBinary:  epicsPath,
		ClaudeBinary: claudePath,
	})
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("daemon shutdown: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for daemon shutdown")
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		client, err := daemon.NewClient(home)
		if err == nil {
			var status map[string]any
			if err := client.Call(context.Background(), "daemon.status", map[string]any{}, &status); err == nil {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("daemon did not become ready")
}

func newShortCLIDir(t *testing.T, prefix string) string {
	t.Helper()
	id, err := daemonstore.GenerateID(prefix + "-")
	if err != nil {
		t.Fatalf("generate id: %v", err)
	}
	path := filepath.Join(os.TempDir(), id)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir short dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func copyDir(src, dest string) error {
	return fsutil.CopyDir(src, dest)
}
