package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidCronExpression(t *testing.T) {
	root := t.TempDir()
	writeCronFixture(t, root, "jobs", "*/5 0 * * 1-5 scripts/run.sh\n")
	writeFile(t, filepath.Join(root, "scripts", "run.sh"), "#!/bin/sh\n")

	diagnostics, err := Validate(root)
	if err != nil {
		t.Fatalf("validate cron: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diagnostics)
	}
}

func TestInvalidCronExpression(t *testing.T) {
	root := t.TempDir()
	writeCronFixture(t, root, "jobs", "60 0 * * * scripts/run.sh\n")

	diagnostics, err := Validate(root)
	if err != nil {
		t.Fatalf("validate cron: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", diagnostics)
	}
	if diagnostics[0].Code != "invalid_cron_expression" {
		t.Fatalf("unexpected diagnostic code: %+v", diagnostics[0])
	}
}

func TestMissingCommand(t *testing.T) {
	root := t.TempDir()
	writeCronFixture(t, root, "jobs", "0 12 * * * scripts/missing.sh\n")

	diagnostics, err := Validate(root)
	if err != nil {
		t.Fatalf("validate cron: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", diagnostics)
	}
	if diagnostics[0].Code != "missing_cron_command" || diagnostics[0].Level != "warning" {
		t.Fatalf("unexpected diagnostic: %+v", diagnostics[0])
	}
}

func TestEmptyCronDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cron.d"), 0o755); err != nil {
		t.Fatalf("mkdir cron dir: %v", err)
	}

	diagnostics, err := Validate(root)
	if err != nil {
		t.Fatalf("validate cron: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diagnostics)
	}
}

func TestNoCronDir(t *testing.T) {
	root := t.TempDir()

	diagnostics, err := Validate(root)
	if err != nil {
		t.Fatalf("validate cron: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diagnostics)
	}
}

func writeCronFixture(t *testing.T, root, name, content string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "cron.d", name), content)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
