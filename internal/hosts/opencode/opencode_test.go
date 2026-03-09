package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeName(t *testing.T) {
	if got := (Adapter{}).Name(); got != "opencode" {
		t.Fatalf("expected opencode, got %q", got)
	}
}

func TestOpenCodeInstallDir(t *testing.T) {
	got := (Adapter{}).InstallDir("/tmp/work", "sample-epic")
	want := filepath.Join("/tmp/work", ".opencode", "skills", "sample-epic")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestOpenCodeSetupCreatesCommandFiles(t *testing.T) {
	dir := t.TempDir()

	result, err := (Adapter{}).Setup(dir)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(result.Created) < 4 {
		t.Fatalf("expected created files, got %+v", result)
	}

	for _, relative := range []string{
		filepath.Join(".opencode", "commands", "epics-resume.md"),
		filepath.Join(".opencode", "commands", "epics-info.md"),
		filepath.Join(".opencode", "commands", "epics-doctor.md"),
		"AGENTS.md",
	} {
		raw, err := os.ReadFile(filepath.Join(dir, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		if strings.TrimSpace(string(raw)) == "" {
			t.Fatalf("expected non-empty content in %s", relative)
		}
	}
}

func TestOpenCodeSetupIsAdditive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("# Existing OpenCode Rules\n"), 0o644); err != nil {
		t.Fatalf("write instructions: %v", err)
	}

	if _, err := (Adapter{}).Setup(dir); err != nil {
		t.Fatalf("setup: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read instructions: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "# Existing OpenCode Rules\n") {
		t.Fatalf("expected original content preserved, got %q", content)
	}
	if !strings.Contains(content, epicGuidanceMarker) {
		t.Fatalf("expected Epic guidance appended, got %q", content)
	}
}

func TestOpenCodeSetupIdempotent(t *testing.T) {
	dir := t.TempDir()
	adapter := Adapter{}

	if _, err := adapter.Setup(dir); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	if _, err := adapter.Setup(dir); err != nil {
		t.Fatalf("second setup: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if strings.Count(string(raw), epicGuidanceMarker) != 1 {
		t.Fatalf("expected one Epic guidance section, got %q", string(raw))
	}
}
