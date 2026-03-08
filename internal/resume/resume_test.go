package resume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/fsutil"
	"github.com/agentepics/epics.sh/internal/testutil"
)

func TestBuildSkipsMissingRecentLogFiles(t *testing.T) {
	root := testutil.RepoRoot(t)
	src := filepath.Join(root, "examples", "fixtures", "resume-epic")
	dest := filepath.Join(t.TempDir(), "resume-epic")

	if err := fsutil.CopyDir(src, dest); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	pkg, diagnostics, err := epic.Validate(dest)
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	if epic.HasErrors(diagnostics) {
		t.Fatalf("expected valid fixture, got diagnostics: %#v", diagnostics)
	}
	if len(pkg.LogFiles) == 0 {
		t.Fatal("expected fixture to contain log files")
	}

	missingLog := pkg.LogFiles[len(pkg.LogFiles)-1]
	if err := os.Remove(missingLog); err != nil {
		t.Fatalf("remove log file: %v", err)
	}

	result, err := Build(pkg)
	if err != nil {
		t.Fatalf("build resume context: %v", err)
	}
	if len(result.LogPaths) != len(pkg.LogFiles)-1 {
		t.Fatalf("expected one fewer log path, got %d", len(result.LogPaths))
	}
}
