package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

func TestGetEntireState(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "state.json"), map[string]any{
		"phase": map[string]any{"current": "planning"},
		"next":  "review",
	})

	value, snapshot, err := Get(dir, "")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	data, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map state, got %T", value)
	}
	if snapshot.Path != filepath.Join(dir, "state.json") {
		t.Fatalf("expected state path, got %s", snapshot.Path)
	}
	if data["next"] != "review" {
		t.Fatalf("expected next=review, got %#v", data["next"])
	}
}

func TestGetNestedKey(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "state.json"), map[string]any{
		"phase": map[string]any{"current": "planning"},
	})

	value, _, err := Get(dir, "phase.current")
	if err != nil {
		t.Fatalf("get nested key: %v", err)
	}
	if value != "planning" {
		t.Fatalf("expected planning, got %#v", value)
	}
}

func TestGetMissingKey(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "state.json"), map[string]any{
		"phase": map[string]any{"current": "planning"},
	})

	if _, _, err := Get(dir, "phase.next"); err == nil {
		t.Fatal("expected missing key to fail")
	}
}

func TestSetCreatesFile(t *testing.T) {
	dir := t.TempDir()

	snapshot, value, err := Set(dir, "phase.current", "planning")
	if err != nil {
		t.Fatalf("set state: %v", err)
	}
	if snapshot.Path != filepath.Join(dir, "state.json") {
		t.Fatalf("expected state.json path, got %s", snapshot.Path)
	}
	if value != "planning" {
		t.Fatalf("expected string value, got %#v", value)
	}

	data := readJSONFile(t, filepath.Join(dir, "state.json"))
	phase := data["phase"].(map[string]any)
	if phase["current"] != "planning" {
		t.Fatalf("expected nested state value, got %#v", phase["current"])
	}
}

func TestSetPreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "state.json"), map[string]any{
		"phase": map[string]any{"current": "planning"},
		"keep":  "yes",
	})

	if _, _, err := Set(dir, "next", `"review"`); err != nil {
		t.Fatalf("set state: %v", err)
	}

	data := readJSONFile(t, filepath.Join(dir, "state.json"))
	if data["keep"] != "yes" {
		t.Fatalf("expected existing field to be preserved, got %#v", data["keep"])
	}
	if data["next"] != "review" {
		t.Fatalf("expected next=review, got %#v", data["next"])
	}
}

func TestSetNestedKey(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := Set(dir, "phase.current.step", `2`); err != nil {
		t.Fatalf("set nested state: %v", err)
	}

	data := readJSONFile(t, filepath.Join(dir, "state.json"))
	phase := data["phase"].(map[string]any)
	current := phase["current"].(map[string]any)
	if current["step"] != float64(2) {
		t.Fatalf("expected numeric nested value, got %#v", current["step"])
	}
}

func TestSetRespectsCorePrecedence(t *testing.T) {
	dir := t.TempDir()
	writeJSONFile(t, filepath.Join(dir, "state.json"), map[string]any{
		"phase": "wrong-file",
	})
	writeJSONFile(t, filepath.Join(dir, "state", "core.json"), map[string]any{
		"phase": "right-file",
	})

	snapshot, _, err := Set(dir, "phase", `"updated"`)
	if err != nil {
		t.Fatalf("set state: %v", err)
	}
	if snapshot.Path != filepath.Join(dir, "state", "core.json") {
		t.Fatalf("expected core state path, got %s", snapshot.Path)
	}

	core := readJSONFile(t, filepath.Join(dir, "state", "core.json"))
	if core["phase"] != "updated" {
		t.Fatalf("expected core state update, got %#v", core["phase"])
	}
	plain := readJSONFile(t, filepath.Join(dir, "state.json"))
	if plain["phase"] != "wrong-file" {
		t.Fatalf("expected state.json to remain unchanged, got %#v", plain["phase"])
	}
}

func TestSetAtomicWrite(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := Set(dir, "phase.current", `"planning"`); err != nil {
		t.Fatalf("set state: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "state.json.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no lingering temp files, got %v", matches)
	}
}

func TestReplaceFileFallsBackOnCrossDeviceRename(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "state.json")
	tempPath := filepath.Join(dir, "state.json.tmp")

	if err := os.WriteFile(destPath, []byte("{\"phase\":\"old\"}\n"), 0o600); err != nil {
		t.Fatalf("write destination: %v", err)
	}
	if err := os.WriteFile(tempPath, []byte("{\"phase\":\"new\"}\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	err := replaceFile(tempPath, destPath, 0o644, func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	})
	if err != nil {
		t.Fatalf("replace file: %v", err)
	}

	raw, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(raw) != "{\"phase\":\"new\"}\n" {
		t.Fatalf("unexpected destination content %q", string(raw))
	}
}

func TestSetConcurrent(t *testing.T) {
	dir := t.TempDir()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "workers.slot" + string(rune('a'+i))
			if _, _, err := Set(dir, key, `"ok"`); err != nil {
				t.Errorf("set concurrent key %s: %v", key, err)
			}
		}()
	}
	wg.Wait()

	data := readJSONFile(t, filepath.Join(dir, "state.json"))
	workers := data["workers"].(map[string]any)
	if len(workers) != 8 {
		t.Fatalf("expected 8 worker entries, got %d", len(workers))
	}
}

func TestSetUsesRuntimeStateForSpec052(t *testing.T) {
	dir := t.TempDir()
	writeRuntimeEpicRoot(t, dir)

	snapshot, _, err := Set(dir, "phase.current", `"planning"`)
	if err != nil {
		t.Fatalf("set runtime state: %v", err)
	}
	expectedPath := filepath.Join(dir, "runtime", "state.json")
	if snapshot.Path != expectedPath {
		t.Fatalf("expected runtime state path %s, got %s", expectedPath, snapshot.Path)
	}

	data := readJSONFile(t, expectedPath)
	phase := data["phase"].(map[string]any)
	if phase["current"] != "planning" {
		t.Fatalf("expected runtime phase.current, got %#v", phase["current"])
	}
}

func writeJSONFile(t *testing.T, path string, data map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read json file: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	return data
}

func writeRuntimeEpicRoot(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "EPIC.md"), []byte("---\nspec_version: 0.5.2\nid: runtime-epic\n---\n\n# Runtime Epic\n"), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}
}
