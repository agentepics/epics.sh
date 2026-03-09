package logutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecentLogs(t *testing.T) {
	root := t.TempDir()
	logDir := filepath.Join(root, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}

	base := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	createLogFile(t, filepath.Join(logDir, "old.md"), "old", base.Add(-2*time.Hour))
	createLogFile(t, filepath.Join(logDir, "newest.md"), "newest", base)
	createLogFile(t, filepath.Join(logDir, "middle.md"), "middle", base.Add(-1*time.Hour))

	entries, err := Recent(root, 3)
	if err != nil {
		t.Fatalf("recent logs: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if filepath.Base(entries[0].Path) != "newest.md" || entries[0].Content != "newest" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if filepath.Base(entries[1].Path) != "middle.md" || entries[1].Content != "middle" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
	if filepath.Base(entries[2].Path) != "old.md" || entries[2].Content != "old" {
		t.Fatalf("unexpected third entry: %+v", entries[2])
	}
}

func TestRecentLogsLimit(t *testing.T) {
	root := t.TempDir()
	logDir := filepath.Join(root, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}

	base := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	createLogFile(t, filepath.Join(logDir, "one.md"), "one", base.Add(-time.Hour))
	createLogFile(t, filepath.Join(logDir, "two.md"), "two", base)

	entries, err := Recent(root, 1)
	if err != nil {
		t.Fatalf("recent logs: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if filepath.Base(entries[0].Path) != "two.md" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestRecentLogsEmpty(t *testing.T) {
	root := t.TempDir()

	entries, err := Recent(root, 3)
	if err != nil {
		t.Fatalf("recent logs: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}

func TestCreateLogWithTitle(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.March, 8, 14, 15, 16, 0, time.UTC)

	path, err := CreateAt(root, "Session 1!", now)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}

	if filepath.Base(path) != "2026-03-08-session-1.md" {
		t.Fatalf("unexpected filename: %s", filepath.Base(path))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "date: 2026-03-08T14:15:16Z") {
		t.Fatalf("missing date in content: %q", content)
	}
	if !strings.Contains(content, "title: Session 1!") {
		t.Fatalf("missing title in content: %q", content)
	}
}

func TestCreateLogWithoutTitle(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.March, 8, 14, 15, 16, 0, time.UTC)

	path, err := CreateAt(root, "", now)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}

	if filepath.Base(path) != "2026-03-08-141516.md" {
		t.Fatalf("unexpected filename: %s", filepath.Base(path))
	}
}

func TestCreateLogDateFormat(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.March, 8, 14, 15, 16, 0, time.UTC)

	path, err := CreateAt(root, "Session", now)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 4 {
		t.Fatalf("unexpected frontmatter: %q", string(raw))
	}

	value := strings.TrimPrefix(lines[1], "date: ")
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		t.Fatalf("expected RFC3339 date, got %q: %v", value, err)
	}
}

func TestCreateLogUsesRuntimeLayoutForSpec052(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("# Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "EPIC.md"), []byte("---\nspec_version: 0.5.2\nid: runtime-epic\n---\n\n# Runtime Epic\n"), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}

	path, err := CreateAt(root, "Session", time.Date(2026, time.March, 8, 14, 15, 16, 0, time.UTC))
	if err != nil {
		t.Fatalf("create runtime log: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(path), "runtime/log/") {
		t.Fatalf("expected runtime log path, got %s", path)
	}
}

func createLogFile(t *testing.T, path, content string, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}
