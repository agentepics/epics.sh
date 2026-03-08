package workspace

import (
	"sync"
	"testing"
)

func TestSaveInstallConcurrentWritersPreserveRecords(t *testing.T) {
	cwd := t.TempDir()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			record := NewInstallRecord(
				"epic-"+string(rune('a'+i)),
				"Epic",
				"claude",
				"./fixture",
				"",
				"",
				".claude/skills/test",
			)
			if err := SaveInstall(cwd, record); err != nil {
				t.Errorf("save install %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	installs, err := LoadInstalls(cwd)
	if err != nil {
		t.Fatalf("load installs: %v", err)
	}
	if len(installs) != 8 {
		t.Fatalf("expected 8 install records, got %d", len(installs))
	}
}
