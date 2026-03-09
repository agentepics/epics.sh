package epic

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadUsesRuntimePathsForSpec052(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: runtime-epic\ndescription: Runtime fixture.\n---\n\n# Runtime Epic\n\nUse this epic when runtime state matters. `EPIC.md` is authoritative.\n\nSee ## Agent Epics below if this is your first encounter with the Agent Epics system.\n\n"+CanonicalSkillFooter())
	writeEpicFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.2\nid: runtime-epic\n---\n\n# Runtime Epic\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "state", "core.json"), "{\n  \"currentPlan\": \"runtime/plans/001-current.md\"\n}\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "plans", "001-current.md"), "# Current\n")
	writeEpicFile(t, filepath.Join(dir, "runtime", "log", "2026-03-08.md"), "# Log\n")

	pkg, err := Load(dir)
	if err != nil {
		t.Fatalf("load package: %v", err)
	}
	if pkg.SpecVersion != "0.5.2" {
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

func TestValidateRejectsLegacyLiveStateForSpec052(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: runtime-epic\ndescription: Runtime fixture.\n---\n\n# Runtime Epic\n\nUse this epic when runtime state matters. `EPIC.md` is authoritative.\n\nSee ## Agent Epics below if this is your first encounter with the Agent Epics system.\n\n"+CanonicalSkillFooter())
	writeEpicFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.2\nid: runtime-epic\n---\n\n# Runtime Epic\n")
	writeEpicFile(t, filepath.Join(dir, "plans", "001-current.md"), "# Legacy Plan\n")

	_, diagnostics, err := Validate(dir)
	if err != nil {
		t.Fatalf("validate package: %v", err)
	}
	if !HasErrors(diagnostics) {
		t.Fatalf("expected legacy live-state path error, got %#v", diagnostics)
	}
}

func TestValidateRequiresDualPurposeSkillSurfaceForSpec052(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "# Runtime Epic\n")
	writeEpicFile(t, filepath.Join(dir, "EPIC.md"), "---\nspec_version: 0.5.2\nid: runtime-epic\n---\n\n# Runtime Epic\n")

	_, diagnostics, err := Validate(dir)
	if err != nil {
		t.Fatalf("validate package: %v", err)
	}

	var codes []string
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	assertContainsCode(t, codes, "missing_skill_frontmatter")
	assertContainsCode(t, codes, "missing_agent_epics_heading")
	assertContainsCode(t, codes, "missing_agent_epics_footer")
}

func TestUpgradeSkillFooterReplacesStaleFooter(t *testing.T) {
	dir := t.TempDir()
	writeEpicFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: runtime-epic\ndescription: Runtime fixture.\n---\n\n# Runtime Epic\n\nUse this epic when runtime state matters. `EPIC.md` is authoritative.\n\n## Agent Epics\n<!-- epics-canonical-footer: https://github.com/agentepics/agentepics/blob/v0.5.1/footer.md -->\n\nOld footer.\n")

	_, changed, err := UpgradeSkillFooter(dir)
	if err != nil {
		t.Fatalf("upgrade skill footer: %v", err)
	}
	if !changed {
		t.Fatal("expected footer refresh to report a change")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read refreshed skill: %v", err)
	}
	content := string(raw)
	if strings.Count(content, CanonicalSkillFooterHeading) != 1 {
		t.Fatalf("expected one footer heading, got %q", content)
	}
	if !strings.Contains(content, CanonicalSkillFooterMarker) {
		t.Fatalf("expected canonical footer marker, got %q", content)
	}
}

func assertContainsCode(t *testing.T, codes []string, want string) {
	t.Helper()
	for _, code := range codes {
		if code == want {
			return
		}
	}
	t.Fatalf("expected diagnostic code %q in %#v", want, codes)
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
