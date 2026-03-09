package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostDoctorClaude(t *testing.T) {
	dir := t.TempDir()
	adapter := Adapter{}

	result, err := adapter.Setup(dir)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(result.Created) == 0 {
		t.Fatal("expected setup to create files")
	}
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "skills", "demo"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "skills", "demo", "SKILL.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	checks, err := adapter.Doctor(dir)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if len(checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(checks))
	}
	for _, check := range checks {
		if check.Status != "ok" {
			t.Fatalf("expected ok check, got %+v", check)
		}
	}
}

func TestHostDoctorMissingDir(t *testing.T) {
	dir := t.TempDir()

	checks, err := Adapter{}.Doctor(dir)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}

	foundFailure := false
	for _, check := range checks {
		if check.Status == "fail" {
			foundFailure = true
			break
		}
	}
	if !foundFailure {
		t.Fatalf("expected at least one failure in %+v", checks)
	}
}

func TestClaudeSetupAppendsGuidance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("write existing instructions: %v", err)
	}

	if _, err := (Adapter{}).Setup(dir); err != nil {
		t.Fatalf("setup: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read instructions: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "# Existing\n") {
		t.Fatalf("expected original content preserved, got %q", content)
	}
	if !strings.Contains(content, epicGuidanceMarker) {
		t.Fatalf("expected Epic guidance appended, got %q", content)
	}
}
