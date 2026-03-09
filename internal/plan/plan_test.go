package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPlans(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, filepath.Join(dir, "plans", "002-next.md"), "# Next Plan\n")
	writePlanFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Current Plan\n")

	plans, err := List(dir)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
	if plans[0].Path != "plans/001-current.md" || plans[0].Title != "Current Plan" {
		t.Fatalf("unexpected first plan: %#v", plans[0])
	}
	if plans[1].Path != "plans/002-next.md" || plans[1].Title != "Next Plan" {
		t.Fatalf("unexpected second plan: %#v", plans[1])
	}
}

func TestListPlansEmpty(t *testing.T) {
	dir := t.TempDir()

	plans, err := List(dir)
	if err != nil {
		t.Fatalf("list empty plans: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("expected no plans, got %d", len(plans))
	}
}

func TestListPlansIgnoresNonMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, filepath.Join(dir, "plans", "002-next.md"), "# Next Plan\n")
	writePlanFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Current Plan\n")
	writeFile(t, filepath.Join(dir, "plans", "README"), "notes\n")
	writeFile(t, filepath.Join(dir, "plans", "010-draft.tmp"), "# Not A Plan\n")

	plans, err := List(dir)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 markdown plans, got %d", len(plans))
	}
}

func TestCurrentPlanFromState(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Current Plan\nbody\n")
	writePlanFile(t, filepath.Join(dir, "plans", "002-next.md"), "# Next Plan\nbody\n")
	writeFile(t, filepath.Join(dir, "state.json"), "{\n  \"plan\": \"plans/001-current.md\"\n}\n")

	entry, content, err := Current(dir)
	if err != nil {
		t.Fatalf("current plan from state: %v", err)
	}
	if entry.Path != "plans/001-current.md" {
		t.Fatalf("expected state-selected plan, got %#v", entry)
	}
	if !strings.Contains(content, "Current Plan") {
		t.Fatalf("expected current plan content, got %q", content)
	}
}

func TestCurrentPlanFallback(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Current Plan\n")
	writePlanFile(t, filepath.Join(dir, "plans", "003-latest.md"), "# Latest Plan\n")
	writePlanFile(t, filepath.Join(dir, "plans", "002-middle.md"), "# Middle Plan\n")

	entry, content, err := Current(dir)
	if err != nil {
		t.Fatalf("current fallback plan: %v", err)
	}
	if entry.Path != "plans/003-latest.md" || entry.Title != "Latest Plan" {
		t.Fatalf("unexpected fallback plan: %#v", entry)
	}
	if !strings.Contains(content, "Latest Plan") {
		t.Fatalf("expected latest content, got %q", content)
	}
}

func TestCreatePlanNumbering(t *testing.T) {
	dir := t.TempDir()
	writePlanFile(t, filepath.Join(dir, "plans", "001-alpha.md"), "# Alpha\n")
	writePlanFile(t, filepath.Join(dir, "plans", "009-beta.md"), "# Beta\n")

	entry, err := Create(dir, "Gamma Plan")
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if entry.Path != "plans/010-gamma-plan.md" {
		t.Fatalf("unexpected plan path: %#v", entry)
	}
}

func TestCreatePlanSlugify(t *testing.T) {
	dir := t.TempDir()

	entry, err := Create(dir, "Initial: Plan / v2!")
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if entry.Path != "plans/001-initial-plan-v2.md" {
		t.Fatalf("unexpected slugified path: %#v", entry)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "plans", "001-initial-plan-v2.md"))
	if err != nil {
		t.Fatalf("read created plan: %v", err)
	}
	if string(raw) != "# Initial: Plan / v2!\n" {
		t.Fatalf("unexpected created content: %q", string(raw))
	}
}

func TestCreatePlanIgnoresNonMarkdownFilesForNumbering(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plans", "099-scratch.tmp"), "notes\n")

	entry, err := Create(dir, "Gamma Plan")
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if entry.Path != "plans/001-gamma-plan.md" {
		t.Fatalf("unexpected plan path: %#v", entry)
	}
}

func TestCreatePlanUsesRuntimeLayoutForSpec051(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "SKILL.md"), "# Skill\n")
	writeFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.1\nid: runtime-epic\n---\n\n# Runtime Epic\n")

	entry, err := Create(dir, "Runtime Plan")
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if entry.Path != "runtime/plans/001-runtime-plan.md" {
		t.Fatalf("unexpected runtime plan path: %#v", entry)
	}
	if _, err := os.Stat(filepath.Join(dir, "runtime", "plans", "001-runtime-plan.md")); err != nil {
		t.Fatalf("expected runtime plan file: %v", err)
	}
}

func writePlanFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir file dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
