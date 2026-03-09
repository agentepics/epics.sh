package daemon

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agentepics/epics.sh/internal/daemon/store"
)

func TestWebhookHMACDedupAndRunLedger(t *testing.T) {
	srv, _, client, cfg, cleanup := startTestServer(t, "0")
	defer cleanup()

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	workspace := registerWorkspace(t, client, workspacePath, "repo-a")
	route := upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "repo-a",
		"preferredAdapter": "claude",
		"authMode":         store.AuthHMAC,
		"hmacHeader":       "X-Hub-Signature-256",
		"secretValue":      "top-secret",
	})

	body := []byte(`{"ok":true}`)
	req, err := http.NewRequest(http.MethodPost, "http://"+cfg.WebhookHTTPAddr+"/v1/webhooks/github/repo-a", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Delivery", "delivery-1")
	req.Header.Set("X-Hub-Signature-256", "sha256="+hmacHex("top-secret", body))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post webhook: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := srv.store.ListRuns(route.ID, workspace.ID, 10)
		if err != nil || len(runs) == 0 {
			return false
		}
		return runs[0].Outcome == store.RunSucceeded
	})

	req2, _ := http.NewRequest(http.MethodPost, "http://"+cfg.WebhookHTTPAddr+"/v1/webhooks/github/repo-a", bytes.NewReader(body))
	req2.Header = req.Header.Clone()
	resp, err = http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("post duplicate webhook: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := srv.store.ListRuns(route.ID, workspace.ID, 10)
		if err != nil || len(runs) < 2 {
			return false
		}
		var sawDeduped bool
		var sawSucceeded bool
		for _, run := range runs {
			if run.Outcome == store.RunDeduped {
				sawDeduped = true
			}
			if run.Outcome == store.RunSucceeded && run.Adapter == "claude" && run.ExecutorID != "" {
				sawSucceeded = true
			}
		}
		return sawDeduped && sawSucceeded
	})
}

func TestTwoWorkspacesIsolatedAndRestartPersistsRecords(t *testing.T) {
	_, home, client, cfg, cleanup := startTestServer(t, "0")
	defer cleanup()

	ws1 := registerWorkspace(t, client, filepath.Join(t.TempDir(), "ws1"), "ws1")
	if err := os.MkdirAll(ws1.Path, 0o755); err != nil {
		t.Fatalf("mkdir ws1: %v", err)
	}
	ws2 := registerWorkspace(t, client, filepath.Join(t.TempDir(), "ws2"), "ws2")
	if err := os.MkdirAll(ws2.Path, 0o755); err != nil {
		t.Fatalf("mkdir ws2: %v", err)
	}

	upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      ws1.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "a",
		"preferredAdapter": "claude",
		"authMode":         store.AuthBearer,
		"secretValue":      "token-a",
	})
	upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      ws2.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "b",
		"preferredAdapter": "claude",
		"authMode":         store.AuthBearer,
		"secretValue":      "token-b",
	})

	postBearerWebhook(t, cfg.WebhookHTTPAddr, "a", "token-a")
	postBearerWebhook(t, cfg.WebhookHTTPAddr, "b", "token-b")

	st := store.Open(home)
	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := st.ListRuns("", "", 10)
		if err != nil || len(runs) < 2 {
			return false
		}
		return runs[0].WorkspaceID != runs[1].WorkspaceID
	})

	cleanup()
	srv2, _, client2, _, cleanup2 := startExistingServer(t, home, "0")
	defer cleanup2()
	var workspaces []store.WorkspaceRecord
	if err := client2.Call(context.Background(), "workspace.list", map[string]any{}, &workspaces); err != nil {
		t.Fatalf("workspace list after restart: %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces after restart, got %d", len(workspaces))
	}
	var routes []store.RouteRecord
	if err := client2.Call(context.Background(), "route.list", map[string]any{}, &routes); err != nil {
		t.Fatalf("route list after restart: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes after restart, got %d", len(routes))
	}
	_ = srv2
}

func TestWorkspaceMissingFailsClosedAndSelectedAdapterFailsClosed(t *testing.T) {
	srv, home, client, cfg, cleanup := startTestServer(t, "0")
	defer cleanup()

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	workspace := registerWorkspace(t, client, workspacePath, "repo-a")
	upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "repo-a",
		"preferredAdapter": "claude",
		"authMode":         store.AuthBearer,
		"secretValue":      "token",
	})

	if err := os.RemoveAll(workspacePath); err != nil {
		t.Fatalf("remove workspace: %v", err)
	}
	status := postBearerWebhook(t, cfg.WebhookHTTPAddr, "repo-a", "token")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for missing workspace, got %d", status)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := srv.store.ListRuns("", workspace.ID, 10)
		if err != nil {
			return false
		}
		for _, run := range runs {
			if run.Outcome == store.RunRejected && run.FailureReason == "workspace_degraded" && run.HTTPStatus == http.StatusServiceUnavailable {
				return true
			}
		}
		return false
	})

	cleanup()
	srv2, _, _, cfg2, cleanup2 := startExistingServerWithClaude(t, home, filepath.Join(t.TempDir(), "missing-claude"), "0")
	defer cleanup2()
	_ = srv2
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("restore workspace: %v", err)
	}
	status = postBearerWebhook(t, cfg2.WebhookHTTPAddr, "repo-a", "token")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for unhealthy selected adapter, got %d", status)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := store.Open(home).ListRuns("", workspace.ID, 20)
		if err != nil {
			return false
		}
		for _, run := range runs {
			if run.Outcome == store.RunRejected && run.FailureReason == "adapter_unavailable" && run.HTTPStatus == http.StatusServiceUnavailable {
				return true
			}
		}
		return false
	})
	_ = srv
}

func TestWebhookAuthRejectionIsLedgered(t *testing.T) {
	srv, _, client, cfg, cleanup := startTestServer(t, "0")
	defer cleanup()

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	workspace := registerWorkspace(t, client, workspacePath, "repo-a")
	_ = upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "repo-a",
		"preferredAdapter": "claude",
		"authMode":         store.AuthBearer,
		"secretValue":      "correct-token",
	})

	status := postBearerWebhook(t, cfg.WebhookHTTPAddr, "repo-a", "wrong-token")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := srv.store.ListRuns("", workspace.ID, 10)
		if err != nil {
			return false
		}
		for _, run := range runs {
			if run.Outcome == store.RunRejected && run.FailureReason == "auth_failed" && run.HTTPStatus == http.StatusUnauthorized {
				return true
			}
		}
		return false
	})
}

func TestQueueSaturationIsLedgered(t *testing.T) {
	srv, _, client, cfg, cleanup := startTestServerWithConfig(t, "1", func(cfg *store.Config) {
		cfg.GlobalQueueCapacity = 1
	})
	defer cleanup()

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	workspace := registerWorkspace(t, client, workspacePath, "repo-a")
	_ = upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeWebhook,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"provider":         "github",
		"endpointKey":      "repo-a",
		"preferredAdapter": "claude",
		"authMode":         store.AuthBearer,
		"secretValue":      "token",
	})

	status := postBearerWebhookDelivery(t, cfg.WebhookHTTPAddr, "repo-a", "token", "delivery-1")
	if status != http.StatusAccepted {
		t.Fatalf("expected first webhook 202, got %d", status)
	}
	status = postBearerWebhookDelivery(t, cfg.WebhookHTTPAddr, "repo-a", "token", "delivery-2")
	if status != http.StatusAccepted {
		t.Fatalf("expected second webhook 202, got %d", status)
	}
	status = postBearerWebhookDelivery(t, cfg.WebhookHTTPAddr, "repo-a", "token", "delivery-3")
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", status)
	}
	waitForCondition(t, 5*time.Second, func() bool {
		runs, err := srv.store.ListRuns("", workspace.ID, 10)
		if err != nil {
			return false
		}
		for _, run := range runs {
			if run.Outcome == store.RunRejected && run.FailureReason == "queue_saturated" && run.HTTPStatus == http.StatusTooManyRequests {
				return true
			}
		}
		return false
	})
}

func TestCronCatchupAndOverlapPolicies(t *testing.T) {
	srv, home, client, _, cleanup := startTestServer(t, "1")
	defer cleanup()

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	workspace := registerWorkspace(t, client, workspacePath, "repo-a")

	st := store.Open(home)
	state, err := st.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.LastSchedulerTickAt = time.Now().Add(-3 * time.Minute).UTC().Format(time.RFC3339)
	if err := st.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	srv.mu.Lock()
	srv.state.LastSchedulerTickAt = state.LastSchedulerTickAt
	srv.mu.Unlock()

	routeQueueOne := upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeCron,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"jobName":          "queue-one",
		"cronExpr":         "* * * * *",
		"preferredAdapter": "claude",
		"authMode":         store.AuthNone,
		"overlapPolicy":    store.OverlapQueueOne,
	})
	routeSingleFlight := upsertRoute(t, client, map[string]any{
		"type":             store.RouteTypeCron,
		"workspaceId":      workspace.ID,
		"epicSlug":         "resume-epic",
		"jobName":          "single-flight",
		"cronExpr":         "* * * * *",
		"preferredAdapter": "claude",
		"authMode":         store.AuthNone,
		"overlapPolicy":    store.OverlapSingleFlight,
	})

	srv.catchUpCron()

	waitForCondition(t, 8*time.Second, func() bool {
		queueOneRuns, err := st.ListRuns(routeQueueOne.ID, workspace.ID, 10)
		if err != nil {
			return false
		}
		singleFlightRuns, err := st.ListRuns(routeSingleFlight.ID, workspace.ID, 10)
		if err != nil {
			return false
		}
		return countOutcomes(queueOneRuns, store.RunSucceeded) >= 2 &&
			countOutcomes(queueOneRuns, store.RunSkipped) >= 1 &&
			countOutcomes(singleFlightRuns, store.RunSucceeded) >= 1 &&
			countOutcomes(singleFlightRuns, store.RunSkipped) >= 1
	})
}

func startTestServer(t *testing.T, claudeSleep string) (*Server, string, *Client, store.Config, func()) {
	t.Helper()
	return startTestServerWithConfig(t, claudeSleep, nil)
}

func startTestServerWithConfig(t *testing.T, claudeSleep string, mutate func(*store.Config)) (*Server, string, *Client, store.Config, func()) {
	t.Helper()
	return startServerWithHome(t, newShortDir(t, "epicsd-home"), claudeSleep, "", mutate)
}

func startExistingServer(t *testing.T, home, claudeSleep string) (*Server, string, *Client, store.Config, func()) {
	t.Helper()
	return startServerWithHome(t, home, claudeSleep, "", nil)
}

func startExistingServerWithClaude(t *testing.T, home, claudePath, claudeSleep string) (*Server, string, *Client, store.Config, func()) {
	t.Helper()
	return startServerWithHome(t, home, claudeSleep, claudePath, nil)
}

func startServerWithHome(t *testing.T, home, claudeSleep, claudeOverride string, mutate func(*store.Config)) (*Server, string, *Client, store.Config, func()) {
	t.Helper()

	binDir := t.TempDir()
	epicsPath := writeExecutable(t, binDir, "epics", "#!/bin/sh\necho \"resume:$2\"\n")
	claudePath := claudeOverride
	if claudePath == "" {
		claudePath = writeExecutable(t, binDir, "claude", "#!/bin/sh\nif [ -n \"$EPICSD_TEST_CLAUDE_SLEEP\" ]; then sleep \"$EPICSD_TEST_CLAUDE_SLEEP\"; fi\npwd >> \"$EPICSD_TEST_CLAUDE_LOG\"\nprintf '%s\\n' \"$*\" >> \"$EPICSD_TEST_CLAUDE_LOG\"\n")
	}
	logPath := filepath.Join(t.TempDir(), "claude.log")
	t.Setenv("EPICSD_TEST_CLAUDE_LOG", logPath)
	t.Setenv("EPICSD_TEST_CLAUDE_SLEEP", claudeSleep)

	st := store.Open(home)
	cfg := store.DefaultConfig(home)
	cfg.WebhookHTTPAddr = "127.0.0.1:0"
	cfg.AdminSocketPath = filepath.Join(home, "epicsd.sock")
	cfg.SchedulerTickSeconds = 1
	cfg.PerWorkspaceConcurrency = 1
	if mutate != nil {
		mutate(&cfg)
	}
	if err := st.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	server, err := New(Options{
		Home:         home,
		EpicsBinary:  epicsPath,
		ClaudeBinary: claudePath,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	waitForCondition(t, 5*time.Second, func() bool {
		client, err := NewClient(home)
		if err != nil {
			return false
		}
		var status map[string]any
		if err := client.Call(context.Background(), "daemon.status", map[string]any{}, &status); err != nil {
			return false
		}
		return status["webhookHTTPAddr"] != ""
	})

	client, err := NewClient(home)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	cfg, err = st.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-errCh:
				if err != nil {
					t.Fatalf("server shutdown: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for daemon shutdown")
			}
		})
	}
	return server, home, client, cfg, cleanup
}

func TestStartHTTPServerRejectsNonLoopbackAddress(t *testing.T) {
	home := newShortDir(t, "epicsd-home")
	st := store.Open(home)
	cfg := store.DefaultConfig(home)
	cfg.WebhookHTTPAddr = "0.0.0.0:42617"
	if err := st.SaveConfig(cfg); err == nil {
		t.Fatal("expected SaveConfig to reject non-loopback address")
	}
}

func TestStartAdminServerDoesNotDeleteDirectories(t *testing.T) {
	home := newShortDir(t, "epicsd-home")
	socketPath := filepath.Join(home, "socket-dir")
	if err := os.MkdirAll(filepath.Join(socketPath, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}

	server := &Server{
		cfg: store.Config{
			AdminSocketPath: socketPath,
		},
	}
	err := server.startAdminServer()
	if err == nil {
		t.Fatal("expected startAdminServer to reject non-socket path")
	}
	if _, statErr := os.Stat(filepath.Join(socketPath, "nested")); statErr != nil {
		t.Fatalf("expected socket directory contents preserved: %v", statErr)
	}
}

func registerWorkspace(t *testing.T, client *Client, path, name string) store.WorkspaceRecord {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir workspace path: %v", err)
	}
	var record store.WorkspaceRecord
	if err := client.Call(context.Background(), "workspace.register", map[string]string{
		"path":        path,
		"displayName": name,
	}, &record); err != nil {
		t.Fatalf("workspace.register: %v", err)
	}
	return record
}

func upsertRoute(t *testing.T, client *Client, payload map[string]any) store.RouteRecord {
	t.Helper()
	var route store.RouteRecord
	if err := client.Call(context.Background(), "route.upsert", payload, &route); err != nil {
		t.Fatalf("route.upsert: %v", err)
	}
	return route
}

func postBearerWebhook(t *testing.T, addr, endpointKey, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/webhooks/github/"+endpointKey, strings.NewReader(`{"ok":true}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post webhook: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func postBearerWebhookDelivery(t *testing.T, addr, endpointKey, token, deliveryID string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/webhooks/github/"+endpointKey, strings.NewReader(`{"ok":true}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post webhook: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func writeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", name, err)
	}
	return path
}

func newShortDir(t *testing.T, prefix string) string {
	t.Helper()
	id, err := store.GenerateID(prefix + "-")
	if err != nil {
		t.Fatalf("generate id: %v", err)
	}
	path := filepath.Join(os.TempDir(), id)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir short dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func hmacHex(secret string, body []byte) string {
	sum := hmac.New(sha256.New, []byte(secret))
	sum.Write(body)
	return hex.EncodeToString(sum.Sum(nil))
}

func countOutcomes(runs []store.RunRecord, outcome string) int {
	count := 0
	for _, run := range runs {
		if run.Outcome == outcome {
			count++
		}
	}
	return count
}
