package epic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterReturnsPartialMapWhenUnclosed(t *testing.T) {
	values := parseFrontmatter("---\nid: partial-epic\nsummary: still-present\n")
	if values["id"] != "partial-epic" {
		t.Fatalf("expected partial id, got %#v", values)
	}
	if values["summary"] != "still-present" {
		t.Fatalf("expected partial summary, got %#v", values)
	}
}

func TestLoadUsesRuntimePathsForSpec051(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "# Skill\n")
	writeEpicFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.1\nid: runtime-epic\n---\n\n# Runtime Epic\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "state", "core.json"), "{\n  \"currentPlan\": \"runtime/plans/001-current.md\"\n}\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "plans", "001-current.md"), "# Current\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "log", "2026-03-08.md"), "# Log\n")

	pkg, err := Load(dir)
	if err != nil {
		t.Fatalf("load package: %v", err)
	}
	if pkg.SpecVersion != "0.5.1" {
		t.Fatalf("unexpected spec version: %q", pkg.SpecVersion)
	}
	if pkg.StateCore != filepath.Join(dir, "runtime", "state", "core.json") {
		t.Fatalf("unexpected core state path: %s", pkg.StateCore)
	}
	if len(pkg.PlanFiles) != 1 || pkg.PlanFiles[0] != filepath.Join(dir, "runtime", "plans", "001-current.md") {
		t.Fatalf("unexpected plan files: %#v", pkg.PlanFiles)
	}
	if len(pkg.LogFiles) != 1 || pkg.LogFiles[0] != filepath.Join(dir, "runtime", "log", "2026-03-08.md") {
		t.Fatalf("unexpected log files: %#v", pkg.LogFiles)
	}
}

func TestValidateRejectsLegacyLiveStateForSpec051(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "# Skill\n")
	writeEpicFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.1\nid: runtime-epic\n---\n\n# Runtime Epic\n")
	writeEpicFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Legacy Plan\n")

	_, diagnostics, err := Validate(dir)
	if err != nil {
		t.Fatalf("validate package: %v", err)
	}
	if !HasErrors(diagnostics) {
		t.Fatalf("expected legacy live-state path error, got %#v", diagnostics)
	}
}

func writeEpicFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
