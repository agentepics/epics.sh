package harness

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "epics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "epics", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != root {
		t.Fatalf("expected %s, got %s", root, found)
	}
}

func TestSelectScenarios(t *testing.T) {
	all := []Scenario{
		{Name: "alpha", Tags: []string{"core"}},
		{Name: "beta", Tags: []string{"host"}},
	}

	selected, err := SelectScenarios(all, []string{"beta"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 || selected[0].Name != "beta" {
		t.Fatalf("unexpected selection: %#v", selected)
	}

	selected, err = SelectScenarios(all, nil, "core")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 || selected[0].Name != "alpha" {
		t.Fatalf("unexpected tag selection: %#v", selected)
	}
}

func TestAssertWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	log, err := newOperationLogger(filepath.Join(dir, "operations.log"))
	if err != nil {
		t.Fatalf("newOperationLogger: %v", err)
	}
	defer log.Close()

	err = assertWorkspace(dir, []FileAssertion{
		{Path: "CLAUDE.md", MustExist: true, Equals: "# Existing\n"},
	}, log, "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintList(t *testing.T) {
	var buf bytes.Buffer
	err := PrintList(&buf, []Scenario{
		{Name: "b", Tags: []string{"host"}},
		{Name: "a", Tags: []string{"core"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "a\tcore") || !strings.Contains(output, "b\thost") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestEventLogPath(t *testing.T) {
	got := eventLogPath("/tmp/operations.log")
	want := "/tmp/operations.events.jsonl"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestSnapshotWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest, err := snapshotWorkspace(dir, manifestPath)
	if err != nil {
		t.Fatalf("snapshotWorkspace: %v", err)
	}
	if len(manifest.Entries) == 0 {
		t.Fatalf("expected manifest entries")
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest file: %v", err)
	}
}

func TestValidateRequiredEnv(t *testing.T) {
	t.Setenv("PRESENT_KEY", "value")

	err := validateRequiredEnv([]Scenario{
		{Name: "alpha", RequiredEnv: []string{"PRESENT_KEY"}},
		{Name: "beta", RequiredEnv: []string{"MISSING_KEY"}},
		{Name: "gamma", RequiredEnv: []string{"MISSING_KEY", "ANOTHER_MISSING_KEY"}},
	})
	if err == nil {
		t.Fatal("expected missing env error")
	}

	message := err.Error()
	if !strings.Contains(message, "MISSING_KEY") || !strings.Contains(message, "ANOTHER_MISSING_KEY") {
		t.Fatalf("expected missing keys in error, got: %s", message)
	}
	if !strings.Contains(message, "beta") || !strings.Contains(message, "gamma") {
		t.Fatalf("expected scenario names in error, got: %s", message)
	}
}
