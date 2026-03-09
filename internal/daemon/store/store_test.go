package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreSaveLoadAndRunList(t *testing.T) {
	dir := t.TempDir()
	st := Open(dir)
	if err := st.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	workspaces := []WorkspaceRecord{{
		ID:          "ws_1",
		Path:        "/tmp/ws",
		DisplayName: "repo",
		Enabled:     true,
		Health:      HealthOK,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}}
	if err := st.SaveWorkspaces(workspaces); err != nil {
		t.Fatalf("save workspaces: %v", err)
	}
	loadedWorkspaces, err := st.LoadWorkspaces()
	if err != nil {
		t.Fatalf("load workspaces: %v", err)
	}
	if len(loadedWorkspaces) != 1 || loadedWorkspaces[0].ID != "ws_1" {
		t.Fatalf("unexpected workspaces: %+v", loadedWorkspaces)
	}

	routes := []RouteRecord{{
		ID:              "webhook:github:test",
		Type:            RouteTypeWebhook,
		WorkspaceID:     "ws_1",
		EpicSlug:        "resume-epic",
		Provider:        "github",
		EndpointKey:     "test",
		Enabled:         true,
		AuthMode:        AuthBearer,
		BearerSecretRef: "secret",
		SelectedAdapter: "claude",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}}
	if err := st.SaveRoutes(routes); err != nil {
		t.Fatalf("save routes: %v", err)
	}
	loadedRoutes, err := st.LoadRoutes()
	if err != nil {
		t.Fatalf("load routes: %v", err)
	}
	if len(loadedRoutes) != 1 || loadedRoutes[0].ID != routes[0].ID {
		t.Fatalf("unexpected routes: %+v", loadedRoutes)
	}

	ref, err := st.WriteSecret(routes[0].ID, "bearer", "top-secret")
	if err != nil {
		t.Fatalf("write secret: %v", err)
	}
	secretPath := filepath.Join(st.SecretsDir(), ref)
	info, err := os.Stat(secretPath)
	if err != nil {
		t.Fatalf("stat secret: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected secret perms 0600, got %o", info.Mode().Perm())
	}

	runs := []RunRecord{
		{
			ID:          "run_older",
			RouteID:     routes[0].ID,
			WorkspaceID: "ws_1",
			EpicSlug:    "resume-epic",
			TriggerType: RouteTypeWebhook,
			Outcome:     RunSucceeded,
			EnqueuedAt:  "2026-03-08T12:00:00Z",
			FinishedAt:  "2026-03-08T12:00:02Z",
		},
		{
			ID:          "run_newer",
			RouteID:     routes[0].ID,
			WorkspaceID: "ws_1",
			EpicSlug:    "resume-epic",
			TriggerType: RouteTypeWebhook,
			Outcome:     RunFailed,
			EnqueuedAt:  "2026-03-09T12:00:00Z",
			FinishedAt:  "2026-03-09T12:00:02Z",
		},
	}
	for _, run := range runs {
		if err := st.AppendRun(run); err != nil {
			t.Fatalf("append run %s: %v", run.ID, err)
		}
	}

	listed, err := st.ListRuns(routes[0].ID, "", 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(listed))
	}
	if listed[0].ID != "run_newer" || listed[1].ID != "run_older" {
		t.Fatalf("unexpected order: %+v", listed)
	}
}

func TestConfigRejectsNonLoopbackWebhookAddress(t *testing.T) {
	dir := t.TempDir()
	st := Open(dir)

	err := st.SaveConfig(Config{
		AdminSocketPath: filepath.Join(dir, "epicsd.sock"),
		WebhookHTTPAddr: "0.0.0.0:42617",
	})
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("expected loopback validation error, got %v", err)
	}

	raw, err := json.Marshal(Config{
		AdminSocketPath: filepath.Join(dir, "epicsd.sock"),
		WebhookHTTPAddr: ":42617",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(st.ConfigPath(), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := st.LoadConfig(); err == nil || !strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("expected load validation error, got %v", err)
	}
}
