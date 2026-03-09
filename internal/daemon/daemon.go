package daemon

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/agentepics/epics.sh/internal/daemon/store"
	"github.com/agentepics/epics.sh/internal/doctor"
)

type RuntimeAdapter interface {
	Name() string
	Healthy(ctx context.Context, workspace store.WorkspaceRecord) (bool, string)
	Dispatch(ctx context.Context, route store.RouteRecord, workspace store.WorkspaceRecord, event store.Event) (DispatchResult, error)
}

type DispatchResult struct {
	Outcome       string `json:"outcome"`
	FailureReason string `json:"failureReason,omitempty"`
	Retryable     bool   `json:"retryable"`
}

type Options struct {
	Home         string
	EpicsBinary  string
	ClaudeBinary string
	Now          func() time.Time
}

type Server struct {
	home         string
	store        *store.Store
	cfg          store.Config
	state        store.State
	epicsBinary  string
	claudeBinary string
	now          func() time.Time

	httpServer *http.Server
	httpLn     net.Listener
	adminLn    net.Listener
	logFile    *os.File
	logger     *log.Logger
	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopWG     sync.WaitGroup
	finishCh   chan finishedRun
	notifyCh   chan struct{}

	mu               sync.Mutex
	workspaces       map[string]store.WorkspaceRecord
	routes           map[string]store.RouteRecord
	pending          []*queuedEvent
	routeRunning     map[string]bool
	workspaceRunning map[string]int
	dedup            map[string]time.Time
	accepting        bool
	shuttingDown     bool
	adapters         map[string]RuntimeAdapter
}

type queuedEvent struct {
	run       store.RunRecord
	route     store.RouteRecord
	workspace store.WorkspaceRecord
	event     store.Event
}

type finishedRun struct {
	run       store.RunRecord
	route     store.RouteRecord
	workspace store.WorkspaceRecord
}

type daemonStatus struct {
	Status             string   `json:"status"`
	StartedAt          string   `json:"startedAt"`
	DegradedSubsystems []string `json:"degradedSubsystems"`
	AdminSocketPath    string   `json:"adminSocketPath"`
	WebhookHTTPAddr    string   `json:"webhookHTTPAddr"`
	Workspaces         int      `json:"workspaces"`
	Routes             int      `json:"routes"`
	Pending            int      `json:"pending"`
}

type logsPayload struct {
	Limit int `json:"limit"`
}

type workspaceRegisterPayload struct {
	Path        string `json:"path"`
	DisplayName string `json:"displayName"`
}

type routeUpsertPayload struct {
	RouteID          string `json:"routeId,omitempty"`
	Type             string `json:"type"`
	WorkspaceID      string `json:"workspaceId"`
	EpicSlug         string `json:"epicSlug"`
	Provider         string `json:"provider,omitempty"`
	EndpointKey      string `json:"endpointKey,omitempty"`
	JobName          string `json:"jobName,omitempty"`
	CronExpr         string `json:"cronExpr,omitempty"`
	PreferredAdapter string `json:"preferredAdapter,omitempty"`
	PinnedAdapter    string `json:"pinnedAdapter,omitempty"`
	ReselectAdapter  bool   `json:"reselectAdapter"`
	AuthMode         string `json:"authMode"`
	HMACHeader       string `json:"hmacHeader,omitempty"`
	OverlapPolicy    string `json:"overlapPolicy,omitempty"`
	SecretValue      string `json:"secretValue,omitempty"`
}

type routeInspectPayload struct {
	RouteID string `json:"routeId"`
}

type runListPayload struct {
	RouteID     string `json:"routeId"`
	WorkspaceID string `json:"workspaceId"`
	Limit       int    `json:"limit"`
}

type runInspectPayload struct {
	RunID string `json:"runId"`
}

type toggleRoutePayload struct {
	RouteID string `json:"routeId"`
}

type inspectWorkspacePayload struct {
	WorkspaceID string `json:"workspaceId"`
}

type claudeRuntimeAdapter struct {
	epicsBinary  string
	claudeBinary string
	logger       *log.Logger
}

func New(options Options) (*Server, error) {
	home := strings.TrimSpace(options.Home)
	if home == "" {
		resolved, err := store.ResolveHome()
		if err != nil {
			return nil, err
		}
		home = resolved
	}
	st := store.Open(home)
	if err := st.Ensure(); err != nil {
		return nil, err
	}
	cfg, err := st.LoadConfig()
	if err != nil {
		return nil, err
	}
	state, err := st.LoadState()
	if err != nil {
		return nil, err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}

	epicsBinary, err := resolveBinary("epics", options.EpicsBinary)
	if err != nil {
		return nil, err
	}
	claudeBinary, err := resolveBinary("claude", options.ClaudeBinary)
	if err != nil {
		return nil, err
	}

	server := &Server{
		home:             home,
		store:            st,
		cfg:              cfg,
		state:            state,
		epicsBinary:      epicsBinary,
		claudeBinary:     claudeBinary,
		now:              now,
		finishCh:         make(chan finishedRun, 64),
		notifyCh:         make(chan struct{}, 1),
		workspaces:       map[string]store.WorkspaceRecord{},
		routes:           map[string]store.RouteRecord{},
		routeRunning:     map[string]bool{},
		workspaceRunning: map[string]int{},
		dedup:            map[string]time.Time{},
		accepting:        true,
	}
	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.openLogger(); err != nil {
		return err
	}
	defer s.closeLogger()

	s.adapters = map[string]RuntimeAdapter{
		"claude": &claudeRuntimeAdapter{
			epicsBinary:  s.epicsBinary,
			claudeBinary: s.claudeBinary,
			logger:       s.logger,
		},
	}

	if err := s.loadRecords(); err != nil {
		return err
	}
	if err := s.recoverState(ctx); err != nil {
		return err
	}
	if err := s.startAdminServer(); err != nil {
		return err
	}
	if err := s.startHTTPServer(); err != nil {
		_ = s.adminLn.Close()
		return err
	}

	s.loopCtx, s.loopCancel = context.WithCancel(context.Background())
	s.loopWG.Add(2)
	go s.dispatchLoop()
	go s.schedulerLoop()

	errCh := make(chan error, 2)
	go func() {
		err := s.httpServer.Serve(s.httpLn)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		err := s.serveAdmin()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-errCh:
		_ = s.shutdown()
		return err
	}
}

func (s *Server) shutdown() error {
	s.mu.Lock()
	s.accepting = false
	s.shuttingDown = true
	s.mu.Unlock()

	if s.adminLn != nil {
		_ = s.adminLn.Close()
	}
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = s.httpServer.Shutdown(shutdownCtx)
		cancel()
	}

	deadline := time.NewTimer(time.Duration(s.cfg.ShutdownTimeoutSeconds) * time.Second)
	defer deadline.Stop()

	for {
		if s.runtimeIdle() {
			break
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-deadline.C:
			s.rejectPending("daemon_shutdown", http.StatusServiceUnavailable)
			if s.loopCancel != nil {
				s.loopCancel()
			}
			s.loopWG.Wait()
			return nil
		}
	}

	if s.loopCancel != nil {
		s.loopCancel()
	}
	s.loopWG.Wait()
	return nil
}

func (s *Server) startAdminServer() error {
	if err := removeStaleSocket(s.cfg.AdminSocketPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.cfg.AdminSocketPath), 0o755); err != nil {
		return err
	}
	ln, err := net.Listen("unix", s.cfg.AdminSocketPath)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.cfg.AdminSocketPath, 0o600); err != nil {
		_ = ln.Close()
		return err
	}
	s.adminLn = ln
	return nil
}

func (s *Server) startHTTPServer() error {
	if err := validateLoopbackWebhookAddr(s.cfg.WebhookHTTPAddr); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", s.handleHealth)
	mux.HandleFunc("/v1/webhooks/", s.handleWebhook)
	ln, err := net.Listen("tcp", s.cfg.WebhookHTTPAddr)
	if err != nil {
		return err
	}
	s.httpLn = ln
	s.cfg.WebhookHTTPAddr = ln.Addr().String()
	if err := s.store.SaveConfig(s.cfg); err != nil {
		_ = ln.Close()
		return err
	}
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return nil
}

func (s *Server) serveAdmin() error {
	for {
		conn, err := s.adminLn.Accept()
		if err != nil {
			return err
		}
		go s.handleAdminConn(conn)
	}
}

func (s *Server) handleAdminConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 2*1024*1024)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = writeResponse(conn, Response{OK: false, Error: &APIError{Code: "invalid_json", Message: err.Error()}})
			continue
		}
		resp := s.handleRequest(req)
		if err := writeResponse(conn, resp); err != nil {
			return
		}
	}
}

func (s *Server) handleRequest(req Request) Response {
	var (
		result any
		err    error
	)

	switch req.Action {
	case "daemon.status":
		result = s.statusResult()
	case "daemon.logs":
		var payload logsPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.readLogs(payload.Limit)
		}
	case "daemon.doctor":
		result, err = s.doctorResult()
	case "workspace.register":
		if s.isShuttingDown() {
			err = &APIError{Code: "daemon_shutting_down", Message: "daemon is shutting down"}
			break
		}
		var payload workspaceRegisterPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.registerWorkspace(payload)
		}
	case "workspace.list":
		result = s.listWorkspaces()
	case "workspace.inspect":
		var payload inspectWorkspacePayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.inspectWorkspace(payload.WorkspaceID)
		}
	case "route.upsert":
		if s.isShuttingDown() {
			err = &APIError{Code: "daemon_shutting_down", Message: "daemon is shutting down"}
			break
		}
		var payload routeUpsertPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.upsertRoute(payload)
		}
	case "route.list":
		result = s.listRoutes()
	case "route.inspect":
		var payload routeInspectPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.inspectRoute(payload.RouteID)
		}
	case "route.enable":
		if s.isShuttingDown() {
			err = &APIError{Code: "daemon_shutting_down", Message: "daemon is shutting down"}
			break
		}
		var payload toggleRoutePayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.setRouteEnabled(payload.RouteID, true)
		}
	case "route.disable":
		if s.isShuttingDown() {
			err = &APIError{Code: "daemon_shutting_down", Message: "daemon is shutting down"}
			break
		}
		var payload toggleRoutePayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.setRouteEnabled(payload.RouteID, false)
		}
	case "run.list":
		var payload runListPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.store.ListRuns(payload.RouteID, payload.WorkspaceID, payload.Limit)
		}
	case "run.inspect":
		var payload runInspectPayload
		if err = decodePayload(req.Payload, &payload); err == nil {
			result, err = s.inspectRun(payload.RunID)
		}
	default:
		err = &APIError{Code: "unknown_action", Message: fmt.Sprintf("unsupported action %q", req.Action)}
	}
	if err != nil {
		apiErr := &APIError{Code: "internal_error", Message: err.Error()}
		if typed, ok := err.(*APIError); ok {
			apiErr = typed
		}
		return Response{OK: false, Error: apiErr}
	}
	raw, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return Response{OK: false, Error: &APIError{Code: "internal_error", Message: marshalErr.Error()}}
	}
	return Response{OK: true, Result: raw}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload := map[string]any{
		"status":             s.state.Status,
		"startedAt":          s.state.StartedAt,
		"degradedSubsystems": s.state.DegradedSubsystems,
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.isShuttingDown() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "daemon shutting down"})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/webhooks/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.NotFound(w, r)
		return
	}
	provider := parts[0]
	endpointKey := parts[1]

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes))
	if err != nil {
		s.appendWebhookRejection(webhookRouteID(provider, endpointKey), "", "", http.StatusRequestEntityTooLarge, "body_too_large", "")
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}

	routeID := webhookRouteID(provider, endpointKey)
	s.mu.Lock()
	route, ok := s.routes[routeID]
	workspace, wsOK := s.workspaces[route.WorkspaceID]
	s.mu.Unlock()
	if !ok || route.Type != store.RouteTypeWebhook || !route.Enabled || !wsOK {
		s.appendWebhookRejection(routeID, "", "", http.StatusNotFound, "unknown_route", "")
		http.NotFound(w, r)
		return
	}

	if ok, reason := s.verifyWebhookAuth(route, r.Header, body); !ok {
		s.appendWebhookRouteRejection(route, http.StatusUnauthorized, "auth_failed", "")
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": reason})
		return
	}

	ws, err := s.refreshWorkspace(workspace.ID)
	if err != nil || ws.Health != store.HealthOK || !ws.Enabled {
		s.appendWebhookRouteRejection(route, http.StatusServiceUnavailable, "workspace_degraded", "")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "workspace degraded"})
		return
	}

	event, err := s.buildWebhookEvent(route, r, body)
	if err != nil {
		s.appendWebhookRouteRejection(route, http.StatusBadRequest, "invalid_event", "")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if event.DedupKey != "" {
		if s.checkDedup(route.ID, event.DedupKey) {
			runID, _ := store.GenerateID("run_")
			_ = s.store.AppendRun(store.RunRecord{
				ID:            runID,
				RouteID:       route.ID,
				WorkspaceID:   route.WorkspaceID,
				EpicSlug:      route.EpicSlug,
				TriggerType:   event.TriggerType,
				DedupKey:      event.DedupKey,
				Outcome:       store.RunDeduped,
				FailureReason: "duplicate_delivery",
				EnqueuedAt:    s.now().UTC().Format(time.RFC3339),
				HTTPStatus:    http.StatusConflict,
			})
			writeJSON(w, http.StatusConflict, map[string]any{"error": "duplicate delivery"})
			return
		}
	}

	if _, err := s.determineAdapter(context.Background(), route, ws, false); err != nil {
		s.appendWebhookRouteRejection(route, http.StatusServiceUnavailable, "adapter_unavailable", event.DedupKey)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
		return
	}

	runID, err := store.GenerateID("run_")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	queued := queuedEvent{
		run: store.RunRecord{
			ID:          runID,
			RouteID:     route.ID,
			WorkspaceID: route.WorkspaceID,
			EpicSlug:    route.EpicSlug,
			TriggerType: event.TriggerType,
			DedupKey:    event.DedupKey,
			Outcome:     store.RunQueued,
			EnqueuedAt:  s.now().UTC().Format(time.RFC3339),
			HTTPStatus:  http.StatusAccepted,
		},
		route:     route,
		workspace: ws,
		event:     event,
	}
	if accepted := s.enqueue(queued); !accepted {
		s.appendWebhookRouteRejection(route, http.StatusTooManyRequests, "queue_saturated", event.DedupKey)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "queue saturated"})
		return
	}
	if event.DedupKey != "" {
		s.markDedup(route.ID, event.DedupKey)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true, "runId": runID})
}

func (s *Server) dispatchLoop() {
	defer s.loopWG.Done()
	for {
		select {
		case <-s.loopCtx.Done():
			return
		case finished := <-s.finishCh:
			s.handleFinished(finished)
		case <-s.notifyCh:
			s.startReadyEvents()
		}
	}
}

func (s *Server) schedulerLoop() {
	defer s.loopWG.Done()
	interval := time.Duration(s.cfg.SchedulerTickSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	s.catchUpCron()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.loopCtx.Done():
			return
		case <-ticker.C:
			s.catchUpCron()
		}
	}
}

func (s *Server) catchUpCron() {
	now := s.now()
	s.mu.Lock()
	lastTick := s.state.LastSchedulerTickAt
	var routes []store.RouteRecord
	for _, route := range s.routes {
		if route.Type == store.RouteTypeCron && route.Enabled {
			routes = append(routes, route)
		}
	}
	s.mu.Unlock()

	var since time.Time
	if parsed, err := time.Parse(time.RFC3339, lastTick); err == nil {
		since = parsed
	} else {
		since = now
	}
	for _, route := range routes {
		ws, err := s.refreshWorkspace(route.WorkspaceID)
		if err != nil || ws.Health != store.HealthOK || !ws.Enabled {
			continue
		}
		for _, due := range cronDueBetween(route.CronExpr, since, now) {
			event := store.Event{
				RouteID:     route.ID,
				TriggerType: store.RouteTypeCron,
				DedupKey:    route.ID + ":" + due.UTC().Format(time.RFC3339),
				OccurredAt:  due.UTC().Format(time.RFC3339),
				Payload:     []byte(`{"scheduledAt":"` + due.UTC().Format(time.RFC3339) + `"}`),
				Metadata: map[string]string{
					"scheduledAt": due.UTC().Format(time.RFC3339),
				},
			}
			_ = s.enqueueCron(route, ws, event)
		}
	}

	s.mu.Lock()
	s.state.LastSchedulerTickAt = now.UTC().Format(time.RFC3339)
	s.mu.Unlock()
	_ = s.saveState()
}

func (s *Server) enqueueCron(route store.RouteRecord, workspace store.WorkspaceRecord, event store.Event) error {
	if _, err := s.determineAdapter(context.Background(), route, workspace, false); err != nil {
		runID, _ := store.GenerateID("run_")
		return s.store.AppendRun(store.RunRecord{
			ID:            runID,
			RouteID:       route.ID,
			WorkspaceID:   route.WorkspaceID,
			EpicSlug:      route.EpicSlug,
			TriggerType:   event.TriggerType,
			DedupKey:      event.DedupKey,
			Outcome:       store.RunRejected,
			FailureReason: err.Error(),
			EnqueuedAt:    s.now().UTC().Format(time.RFC3339),
			HTTPStatus:    http.StatusServiceUnavailable,
		})
	}
	runID, err := store.GenerateID("run_")
	if err != nil {
		return err
	}
	queued := queuedEvent{
		run: store.RunRecord{
			ID:          runID,
			RouteID:     route.ID,
			WorkspaceID: route.WorkspaceID,
			EpicSlug:    route.EpicSlug,
			TriggerType: event.TriggerType,
			DedupKey:    event.DedupKey,
			Outcome:     store.RunQueued,
			EnqueuedAt:  s.now().UTC().Format(time.RFC3339),
		},
		route:     route,
		workspace: workspace,
		event:     event,
	}
	ok, skipped := s.enqueueCronWithPolicy(queued)
	if skipped {
		return s.store.AppendRun(store.RunRecord{
			ID:            runID,
			RouteID:       route.ID,
			WorkspaceID:   route.WorkspaceID,
			EpicSlug:      route.EpicSlug,
			TriggerType:   event.TriggerType,
			DedupKey:      event.DedupKey,
			Outcome:       store.RunSkipped,
			FailureReason: "cron_overlap",
			EnqueuedAt:    queued.run.EnqueuedAt,
		})
	}
	if !ok {
		return s.store.AppendRun(store.RunRecord{
			ID:            runID,
			RouteID:       route.ID,
			WorkspaceID:   route.WorkspaceID,
			EpicSlug:      route.EpicSlug,
			TriggerType:   event.TriggerType,
			DedupKey:      event.DedupKey,
			Outcome:       store.RunRejected,
			FailureReason: "queue_saturated",
			EnqueuedAt:    queued.run.EnqueuedAt,
		})
	}
	return nil
}

func (s *Server) enqueueCronWithPolicy(event queuedEvent) (accepted bool, skipped bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting {
		return false, false
	}
	routeBusy := s.routeRunning[event.route.ID]
	pendingCount := s.pendingCountForRouteLocked(event.route.ID)
	switch event.route.OverlapPolicy {
	case store.OverlapSingleFlight:
		if routeBusy || pendingCount > 0 {
			return false, true
		}
	case store.OverlapQueueOne:
		if routeBusy && pendingCount >= 1 {
			return false, true
		}
		if pendingCount >= 1 {
			return false, true
		}
	default:
		if routeBusy || pendingCount > 0 {
			return false, true
		}
	}
	if len(s.pending) >= s.cfg.GlobalQueueCapacity {
		return false, false
	}
	s.pending = append(s.pending, &event)
	s.signalWork()
	return true, false
}

func (s *Server) enqueue(event queuedEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting || len(s.pending) >= s.cfg.GlobalQueueCapacity {
		return false
	}
	s.pending = append(s.pending, &event)
	s.signalWork()
	return true
}

func (s *Server) startReadyEvents() {
	for {
		event := s.nextReadyEvent()
		if event == nil {
			return
		}
		s.startEvent(event)
	}
}

func (s *Server) nextReadyEvent() *queuedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneDedupLocked()
	for index, event := range s.pending {
		if s.routeRunning[event.route.ID] {
			continue
		}
		if s.workspaceRunning[event.workspace.ID] >= s.cfg.PerWorkspaceConcurrency {
			continue
		}
		s.pending = append(s.pending[:index], s.pending[index+1:]...)
		s.routeRunning[event.route.ID] = true
		s.workspaceRunning[event.workspace.ID]++
		return event
	}
	return nil
}

func (s *Server) startEvent(event *queuedEvent) {
	go func() {
		run := event.run
		ws, err := s.refreshWorkspace(event.workspace.ID)
		if err != nil {
			run.Outcome = store.RunRejected
			run.FailureReason = err.Error()
			run.FinishedAt = s.now().UTC().Format(time.RFC3339)
			s.finishCh <- finishedRun{run: run, route: event.route, workspace: event.workspace}
			return
		}
		event.workspace = ws

		adapterName, err := s.determineAdapter(s.loopCtx, event.route, ws, false)
		if err != nil {
			run.Outcome = store.RunRejected
			run.FailureReason = err.Error()
			run.FinishedAt = s.now().UTC().Format(time.RFC3339)
			s.finishCh <- finishedRun{run: run, route: event.route, workspace: ws}
			return
		}
		run.Adapter = adapterName
		executorID, _ := store.GenerateID("ex_")
		run.ExecutorID = executorID
		run.StartedAt = s.now().UTC().Format(time.RFC3339)
		run.Outcome = store.RunRunning

		result, dispatchErr := s.adapters[adapterName].Dispatch(s.loopCtx, event.route, ws, event.event)
		run.FinishedAt = s.now().UTC().Format(time.RFC3339)
		if dispatchErr != nil {
			run.Outcome = store.RunFailed
			run.FailureReason = dispatchErr.Error()
		} else {
			run.Outcome = result.Outcome
			run.FailureReason = result.FailureReason
		}
		if run.Outcome == "" || run.Outcome == store.RunRunning || run.Outcome == store.RunQueued {
			run.Outcome = store.RunFailed
		}
		s.finishCh <- finishedRun{run: run, route: event.route, workspace: ws}
	}()
}

func (s *Server) handleFinished(finished finishedRun) {
	s.mu.Lock()
	delete(s.routeRunning, finished.route.ID)
	if current := s.workspaceRunning[finished.workspace.ID]; current > 0 {
		s.workspaceRunning[finished.workspace.ID] = current - 1
	}
	route := s.routes[finished.route.ID]
	now := s.now().UTC().Format(time.RFC3339)
	route.LastDeliveryAt = now
	switch finished.run.Outcome {
	case store.RunSucceeded:
		route.LastSuccessAt = now
		route.LastErrorAt = ""
		route.LastErrorMessage = ""
	default:
		route.LastErrorAt = now
		route.LastErrorMessage = finished.run.FailureReason
	}
	route.UpdatedAt = now
	s.routes[route.ID] = route
	s.mu.Unlock()

	_ = s.store.AppendRun(finished.run)
	_ = s.saveRoutes()
	s.signalWork()
}

func (s *Server) loadRecords() error {
	workspaces, err := s.store.LoadWorkspaces()
	if err != nil {
		return err
	}
	routes, err := s.store.LoadRoutes()
	if err != nil {
		return err
	}
	s.mu.Lock()
	for _, ws := range workspaces {
		s.workspaces[ws.ID] = ws
	}
	for _, route := range routes {
		s.routes[route.ID] = route
	}
	s.mu.Unlock()
	return nil
}

func (s *Server) recoverState(ctx context.Context) error {
	s.mu.Lock()
	workspaceIDs := make([]string, 0, len(s.workspaces))
	for id := range s.workspaces {
		workspaceIDs = append(workspaceIDs, id)
	}
	s.mu.Unlock()

	for _, id := range workspaceIDs {
		if _, err := s.refreshWorkspace(id); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.state.StartedAt = s.now().UTC().Format(time.RFC3339)
	s.state.Status = store.HealthOK
	s.state.DegradedSubsystems = []string{}
	for _, ws := range s.workspaces {
		if ws.Health != store.HealthOK {
			s.state.Status = store.HealthDegraded
			s.state.DegradedSubsystems = append(s.state.DegradedSubsystems, "workspaces")
			break
		}
	}
	if s.state.LastSchedulerTickAt == "" {
		s.state.LastSchedulerTickAt = s.now().UTC().Format(time.RFC3339)
	}
	s.mu.Unlock()
	return s.saveState()
}

func (s *Server) registerWorkspace(payload workspaceRegisterPayload) (store.WorkspaceRecord, error) {
	if strings.TrimSpace(payload.Path) == "" {
		return store.WorkspaceRecord{}, &APIError{Code: "invalid_workspace", Message: "workspace path is required"}
	}
	abs, err := filepath.Abs(payload.Path)
	if err != nil {
		return store.WorkspaceRecord{}, err
	}
	abs = filepath.Clean(abs)
	name := strings.TrimSpace(payload.DisplayName)
	if name == "" {
		name = filepath.Base(abs)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.workspaces {
		if existing.Path == abs {
			existing.DisplayName = name
			existing.Enabled = true
			existing.UpdatedAt = s.now().UTC().Format(time.RFC3339)
			refreshed := assessWorkspace(existing, s.now)
			s.workspaces[existing.ID] = refreshed
			if err := s.saveWorkspacesLocked(); err != nil {
				return store.WorkspaceRecord{}, err
			}
			return refreshed, nil
		}
	}

	id, err := store.GenerateID("ws_")
	if err != nil {
		return store.WorkspaceRecord{}, err
	}
	record := store.WorkspaceRecord{
		ID:          id,
		Path:        abs,
		DisplayName: name,
		Enabled:     true,
		CreatedAt:   s.now().UTC().Format(time.RFC3339),
		UpdatedAt:   s.now().UTC().Format(time.RFC3339),
	}
	record = assessWorkspace(record, s.now)
	s.workspaces[record.ID] = record
	if err := s.saveWorkspacesLocked(); err != nil {
		return store.WorkspaceRecord{}, err
	}
	return record, nil
}

func (s *Server) refreshWorkspace(id string) (store.WorkspaceRecord, error) {
	s.mu.Lock()
	record, ok := s.workspaces[id]
	s.mu.Unlock()
	if !ok {
		return store.WorkspaceRecord{}, &APIError{Code: "workspace_not_found", Message: fmt.Sprintf("workspace %s not found", id)}
	}
	record = assessWorkspace(record, s.now)
	s.mu.Lock()
	s.workspaces[id] = record
	s.mu.Unlock()
	if err := s.saveWorkspaces(); err != nil {
		return store.WorkspaceRecord{}, err
	}
	return record, nil
}

func (s *Server) listWorkspaces() []store.WorkspaceRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]store.WorkspaceRecord, 0, len(s.workspaces))
	for _, ws := range s.workspaces {
		items = append(items, ws)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *Server) inspectWorkspace(id string) (store.WorkspaceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.workspaces[id]
	if !ok {
		return store.WorkspaceRecord{}, &APIError{Code: "workspace_not_found", Message: fmt.Sprintf("workspace %s not found", id)}
	}
	return record, nil
}

func (s *Server) upsertRoute(payload routeUpsertPayload) (store.RouteRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.workspaces[payload.WorkspaceID]; !ok {
		return store.RouteRecord{}, &APIError{Code: "workspace_not_found", Message: fmt.Sprintf("workspace %s not found", payload.WorkspaceID)}
	}
	if strings.TrimSpace(payload.EpicSlug) == "" {
		return store.RouteRecord{}, &APIError{Code: "invalid_route", Message: "epicSlug is required"}
	}

	var routeID string
	switch payload.Type {
	case store.RouteTypeWebhook:
		if strings.TrimSpace(payload.Provider) == "" || strings.TrimSpace(payload.EndpointKey) == "" {
			return store.RouteRecord{}, &APIError{Code: "invalid_route", Message: "provider and endpointKey are required"}
		}
		routeID = webhookRouteID(payload.Provider, payload.EndpointKey)
	case store.RouteTypeCron:
		if strings.TrimSpace(payload.JobName) == "" || strings.TrimSpace(payload.CronExpr) == "" {
			return store.RouteRecord{}, &APIError{Code: "invalid_route", Message: "jobName and cronExpr are required"}
		}
		if fields, _ := cronFields(payload.CronExpr); len(fields) == 0 {
			return store.RouteRecord{}, &APIError{Code: "invalid_route", Message: "cronExpr must contain five or six fields"}
		}
		routeID = cronRouteID(payload.WorkspaceID, payload.JobName)
	default:
		return store.RouteRecord{}, &APIError{Code: "invalid_route", Message: fmt.Sprintf("unsupported route type %q", payload.Type)}
	}

	existing, hasExisting := s.routes[routeID]
	if payload.RouteID != "" && payload.RouteID != routeID {
		existing, hasExisting = s.routes[payload.RouteID]
		if !hasExisting {
			return store.RouteRecord{}, &APIError{Code: "route_not_found", Message: fmt.Sprintf("route %s not found", payload.RouteID)}
		}
		delete(s.routes, payload.RouteID)
	}

	now := s.now().UTC().Format(time.RFC3339)
	record := store.RouteRecord{
		ID:               routeID,
		Type:             payload.Type,
		WorkspaceID:      payload.WorkspaceID,
		EpicSlug:         payload.EpicSlug,
		Provider:         payload.Provider,
		EndpointKey:      payload.EndpointKey,
		JobName:          payload.JobName,
		CronExpr:         payload.CronExpr,
		PreferredAdapter: payload.PreferredAdapter,
		PinnedAdapter:    payload.PinnedAdapter,
		Enabled:          true,
		OverlapPolicy:    payload.OverlapPolicy,
		AuthMode:         payload.AuthMode,
		HMACHeader:       payload.HMACHeader,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if hasExisting {
		record.CreatedAt = existing.CreatedAt
		record.Enabled = existing.Enabled
		record.SelectedAdapter = existing.SelectedAdapter
		record.LastDeliveryAt = existing.LastDeliveryAt
		record.LastSuccessAt = existing.LastSuccessAt
		record.LastErrorAt = existing.LastErrorAt
		record.LastErrorMessage = existing.LastErrorMessage
		record.BearerSecretRef = existing.BearerSecretRef
		record.HMACSecretRef = existing.HMACSecretRef
	}
	if record.AuthMode == "" {
		record.AuthMode = store.AuthNone
	}
	if record.Type == store.RouteTypeCron && record.OverlapPolicy == "" {
		record.OverlapPolicy = store.OverlapSingleFlight
	}

	if err := s.validateAdapterNames(record); err != nil {
		return store.RouteRecord{}, err
	}

	if err := s.applySecretLocked(&record, existing, hasExisting, payload); err != nil {
		return store.RouteRecord{}, err
	}

	workspace := assessWorkspace(s.workspaces[payload.WorkspaceID], s.now)
	s.workspaces[payload.WorkspaceID] = workspace
	selected, err := s.selectAdapterLocked(record, workspace, payload.ReselectAdapter, hasExisting)
	if err != nil {
		return store.RouteRecord{}, err
	}
	record.SelectedAdapter = selected

	s.routes[record.ID] = record
	if err := s.saveRoutesLocked(); err != nil {
		return store.RouteRecord{}, err
	}
	if err := s.saveWorkspacesLocked(); err != nil {
		return store.RouteRecord{}, err
	}
	return record, nil
}

func (s *Server) applySecretLocked(record *store.RouteRecord, existing store.RouteRecord, hasExisting bool, payload routeUpsertPayload) error {
	switch record.AuthMode {
	case store.AuthNone:
		_ = s.store.RemoveSecret(existing.BearerSecretRef)
		_ = s.store.RemoveSecret(existing.HMACSecretRef)
		record.BearerSecretRef = ""
		record.HMACSecretRef = ""
	case store.AuthBearer:
		_ = s.store.RemoveSecret(existing.HMACSecretRef)
		if payload.SecretValue != "" {
			ref, err := s.store.WriteSecret(record.ID, "bearer", payload.SecretValue)
			if err != nil {
				return err
			}
			record.BearerSecretRef = ref
		} else if hasExisting && existing.BearerSecretRef != "" {
			value, err := s.store.ReadSecret(existing.BearerSecretRef)
			if err != nil {
				return err
			}
			ref, err := s.store.WriteSecret(record.ID, "bearer", value)
			if err != nil {
				return err
			}
			record.BearerSecretRef = ref
		} else {
			return &APIError{Code: "invalid_route", Message: "bearer routes require secretValue"}
		}
		record.HMACSecretRef = ""
	case store.AuthHMAC:
		if strings.TrimSpace(record.HMACHeader) == "" {
			return &APIError{Code: "invalid_route", Message: "hmacHeader is required for hmac routes"}
		}
		_ = s.store.RemoveSecret(existing.BearerSecretRef)
		if payload.SecretValue != "" {
			ref, err := s.store.WriteSecret(record.ID, "hmac", payload.SecretValue)
			if err != nil {
				return err
			}
			record.HMACSecretRef = ref
		} else if hasExisting && existing.HMACSecretRef != "" {
			value, err := s.store.ReadSecret(existing.HMACSecretRef)
			if err != nil {
				return err
			}
			ref, err := s.store.WriteSecret(record.ID, "hmac", value)
			if err != nil {
				return err
			}
			record.HMACSecretRef = ref
		} else {
			return &APIError{Code: "invalid_route", Message: "hmac routes require secretValue"}
		}
		record.BearerSecretRef = ""
	default:
		return &APIError{Code: "invalid_route", Message: fmt.Sprintf("unsupported authMode %q", record.AuthMode)}
	}
	return nil
}

func (s *Server) selectAdapterLocked(route store.RouteRecord, workspace store.WorkspaceRecord, reselect bool, hasExisting bool) (string, error) {
	if route.PinnedAdapter != "" {
		adapter := s.adapters[route.PinnedAdapter]
		ok, reason := adapter.Healthy(context.Background(), workspace)
		if !ok {
			return "", &APIError{Code: "adapter_unhealthy", Message: reason}
		}
		return route.PinnedAdapter, nil
	}
	if hasExisting && !reselect && route.SelectedAdapter != "" {
		adapter := s.adapters[route.SelectedAdapter]
		ok, reason := adapter.Healthy(context.Background(), workspace)
		if !ok {
			return "", &APIError{Code: "adapter_unhealthy", Message: reason}
		}
		return route.SelectedAdapter, nil
	}
	if route.PreferredAdapter != "" {
		adapter := s.adapters[route.PreferredAdapter]
		ok, reason := adapter.Healthy(context.Background(), workspace)
		if !ok {
			return "", &APIError{Code: "adapter_unhealthy", Message: reason}
		}
		return route.PreferredAdapter, nil
	}
	for name, adapter := range s.adapters {
		ok, _ := adapter.Healthy(context.Background(), workspace)
		if ok {
			return name, nil
		}
	}
	return "", &APIError{Code: "adapter_unhealthy", Message: "no healthy runtime adapters available"}
}

func (s *Server) listRoutes() []store.RouteRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]store.RouteRecord, 0, len(s.routes))
	for _, route := range s.routes {
		items = append(items, route)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *Server) inspectRoute(id string) (store.RouteRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[id]
	if !ok {
		return store.RouteRecord{}, &APIError{Code: "route_not_found", Message: fmt.Sprintf("route %s not found", id)}
	}
	return route, nil
}

func (s *Server) setRouteEnabled(id string, enabled bool) (store.RouteRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[id]
	if !ok {
		return store.RouteRecord{}, &APIError{Code: "route_not_found", Message: fmt.Sprintf("route %s not found", id)}
	}
	route.Enabled = enabled
	route.UpdatedAt = s.now().UTC().Format(time.RFC3339)
	s.routes[id] = route
	if err := s.saveRoutesLocked(); err != nil {
		return store.RouteRecord{}, err
	}
	return route, nil
}

func (s *Server) inspectRun(id string) (store.RunRecord, error) {
	record, ok, err := s.store.InspectRun(id)
	if err != nil {
		return store.RunRecord{}, err
	}
	if !ok {
		return store.RunRecord{}, &APIError{Code: "run_not_found", Message: fmt.Sprintf("run %s not found", id)}
	}
	return record, nil
}

func (s *Server) determineAdapter(ctx context.Context, route store.RouteRecord, workspace store.WorkspaceRecord, reselect bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selectAdapterLocked(route, workspace, reselect, route.SelectedAdapter != "")
}

func (s *Server) validateAdapterNames(route store.RouteRecord) error {
	for _, name := range []string{route.PreferredAdapter, route.PinnedAdapter} {
		if name == "" {
			continue
		}
		if _, ok := s.adapters[name]; !ok {
			return &APIError{Code: "invalid_route", Message: fmt.Sprintf("unknown adapter %q", name)}
		}
	}
	return nil
}

func (s *Server) verifyWebhookAuth(route store.RouteRecord, headers http.Header, body []byte) (bool, string) {
	switch route.AuthMode {
	case store.AuthNone:
		if s.cfg.AllowInsecureAuthNone {
			return true, ""
		}
		return false, "authMode none is disabled"
	case store.AuthBearer:
		secret, err := s.store.ReadSecret(route.BearerSecretRef)
		if err != nil {
			return false, "bearer secret missing"
		}
		authHeader := strings.TrimSpace(headers.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			authHeader = strings.TrimSpace(authHeader[7:])
		}
		if !hmac.Equal([]byte(authHeader), []byte(secret)) {
			return false, "invalid bearer token"
		}
		return true, ""
	case store.AuthHMAC:
		secret, err := s.store.ReadSecret(route.HMACSecretRef)
		if err != nil {
			return false, "hmac secret missing"
		}
		signature := strings.TrimSpace(headers.Get(route.HMACHeader))
		if signature == "" {
			return false, "missing hmac header"
		}
		expectedMAC := hmac.New(sha256.New, []byte(secret))
		expectedMAC.Write(body)
		expected := hex.EncodeToString(expectedMAC.Sum(nil))
		signature = strings.TrimPrefix(signature, "sha256=")
		if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
			return false, "invalid hmac signature"
		}
		return true, ""
	default:
		return false, "unsupported auth mode"
	}
}

func (s *Server) buildWebhookEvent(route store.RouteRecord, r *http.Request, body []byte) (store.Event, error) {
	headers := map[string]string{}
	for key, values := range r.Header {
		if len(values) == 0 {
			continue
		}
		headers[strings.ToLower(key)] = values[0]
	}
	externalID := firstNonEmpty(
		r.Header.Get("X-GitHub-Delivery"),
		r.Header.Get("X-Request-Id"),
		r.Header.Get("Idempotency-Key"),
	)
	dedupKey := externalID
	if dedupKey == "" {
		sum := sha256.Sum256(body)
		dedupKey = hex.EncodeToString(sum[:])
	}
	if !json.Valid(body) {
		body, _ = json.Marshal(map[string]string{"body": string(body)})
	}
	return store.Event{
		RouteID:     route.ID,
		TriggerType: store.RouteTypeWebhook,
		Provider:    route.Provider,
		ExternalID:  externalID,
		DedupKey:    route.ID + ":" + dedupKey,
		OccurredAt:  s.now().UTC().Format(time.RFC3339),
		Payload:     body,
		Headers:     headers,
		Metadata: map[string]string{
			"endpointKey": route.EndpointKey,
		},
	}, nil
}

func (s *Server) statusResult() daemonStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return daemonStatus{
		Status:             s.state.Status,
		StartedAt:          s.state.StartedAt,
		DegradedSubsystems: append([]string(nil), s.state.DegradedSubsystems...),
		AdminSocketPath:    s.cfg.AdminSocketPath,
		WebhookHTTPAddr:    s.cfg.WebhookHTTPAddr,
		Workspaces:         len(s.workspaces),
		Routes:             len(s.routes),
		Pending:            len(s.pending),
	}
}

func (s *Server) doctorResult() (doctor.Result, error) {
	checks := []doctor.Check{
		{Name: "admin-socket", Status: statusFromBool(s.adminLn != nil), Message: s.cfg.AdminSocketPath},
		{Name: "store", Status: "ok", Message: s.home},
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + s.cfg.WebhookHTTPAddr + "/v1/health")
	if err != nil {
		checks = append(checks, doctor.Check{Name: "webhook-listener", Status: "fail", Message: err.Error()})
	} else {
		_ = resp.Body.Close()
		checks = append(checks, doctor.Check{Name: "webhook-listener", Status: "ok", Message: s.cfg.WebhookHTTPAddr})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	unhealthy := 0
	for _, ws := range s.workspaces {
		if ws.Health != store.HealthOK {
			unhealthy++
		}
	}
	if unhealthy > 0 {
		checks = append(checks, doctor.Check{Name: "workspaces", Status: "warning", Message: fmt.Sprintf("%d workspace(s) degraded", unhealthy)})
	} else {
		checks = append(checks, doctor.Check{Name: "workspaces", Status: "ok", Message: fmt.Sprintf("%d workspace(s) healthy", len(s.workspaces))})
	}
	return doctor.Result{Checks: checks}, nil
}

func (s *Server) readLogs(limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	raw, err := os.ReadFile(s.store.LogPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{"path": s.store.LogPath(), "lines": []string{}}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return map[string]any{"path": s.store.LogPath(), "lines": lines}, nil
}

func (s *Server) openLogger() error {
	file, err := os.OpenFile(s.store.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	s.logFile = file
	s.logger = log.New(file, "", log.LstdFlags|log.LUTC)
	return nil
}

func (s *Server) closeLogger() {
	if s.adminLn != nil {
		_ = os.Remove(s.cfg.AdminSocketPath)
	}
	if s.logFile != nil {
		_ = s.logFile.Close()
	}
}

func (a *claudeRuntimeAdapter) Name() string { return "claude" }

func (a *claudeRuntimeAdapter) Healthy(ctx context.Context, workspace store.WorkspaceRecord) (bool, string) {
	if !workspace.Enabled {
		return false, "workspace disabled"
	}
	if workspace.Health != store.HealthOK {
		return false, firstNonEmpty(workspace.HealthMessage, "workspace degraded")
	}
	if _, err := os.Stat(workspace.Path); err != nil {
		return false, err.Error()
	}
	if _, err := exec.LookPath(a.epicsBinary); err != nil {
		return false, "epics binary not found"
	}
	if _, err := exec.LookPath(a.claudeBinary); err != nil {
		return false, "claude binary not found"
	}
	return true, ""
}

func (a *claudeRuntimeAdapter) Dispatch(ctx context.Context, route store.RouteRecord, workspace store.WorkspaceRecord, event store.Event) (DispatchResult, error) {
	resumeCmd := exec.CommandContext(ctx, a.epicsBinary, "resume", route.EpicSlug)
	resumeCmd.Dir = workspace.Path
	resumeOutput, err := resumeCmd.CombinedOutput()
	if err != nil {
		return DispatchResult{Outcome: store.RunFailed, FailureReason: strings.TrimSpace(string(resumeOutput))}, fmt.Errorf("epics resume failed: %w", err)
	}

	prompt := buildClaudePrompt(route, event, string(resumeOutput))
	claudeCmd := exec.CommandContext(ctx, a.claudeBinary, "-p", prompt)
	claudeCmd.Dir = workspace.Path
	output, err := claudeCmd.CombinedOutput()
	if a.logger != nil {
		a.logger.Printf("route=%s adapter=claude output=%s", route.ID, strings.TrimSpace(string(output)))
	}
	if err != nil {
		return DispatchResult{Outcome: store.RunFailed, FailureReason: strings.TrimSpace(string(output))}, fmt.Errorf("claude dispatch failed: %w", err)
	}
	return DispatchResult{Outcome: store.RunSucceeded}, nil
}

func buildClaudePrompt(route store.RouteRecord, event store.Event, resumeContext string) string {
	body := bytes.TrimSpace(event.Payload)
	return strings.TrimSpace(fmt.Sprintf(`
Epic trigger execution

Route ID: %s
Route Type: %s
Epic Slug: %s
Occurred At: %s

Trigger metadata:
%s

Payload:
%s

Resume context:
%s
`, route.ID, route.Type, route.EpicSlug, event.OccurredAt, marshalForPrompt(event.Metadata), string(body), strings.TrimSpace(resumeContext)))
}

func marshalForPrompt(value any) string {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}

func webhookRouteID(provider, endpointKey string) string {
	return "webhook:" + strings.TrimSpace(provider) + ":" + strings.TrimSpace(endpointKey)
}

func cronRouteID(workspaceID, jobName string) string {
	return "cron:" + strings.TrimSpace(workspaceID) + ":" + strings.TrimSpace(jobName)
}

func assessWorkspace(record store.WorkspaceRecord, now func() time.Time) store.WorkspaceRecord {
	record.LastScannedAt = now().UTC().Format(time.RFC3339)
	record.UpdatedAt = now().UTC().Format(time.RFC3339)
	info, err := os.Stat(record.Path)
	switch {
	case err == nil && info.IsDir():
		record.Health = store.HealthOK
		record.HealthMessage = ""
	case err == nil:
		record.Health = store.HealthFail
		record.HealthMessage = "workspace path is not a directory"
	default:
		record.Health = store.HealthFail
		record.HealthMessage = err.Error()
	}
	return record
}

func (s *Server) saveWorkspaces() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveWorkspacesLocked()
}

func (s *Server) saveWorkspacesLocked() error {
	items := make([]store.WorkspaceRecord, 0, len(s.workspaces))
	for _, record := range s.workspaces {
		items = append(items, record)
	}
	return s.store.SaveWorkspaces(items)
}

func (s *Server) saveRoutes() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveRoutesLocked()
}

func (s *Server) saveRoutesLocked() error {
	items := make([]store.RouteRecord, 0, len(s.routes))
	for _, record := range s.routes {
		items = append(items, record)
	}
	return s.store.SaveRoutes(items)
}

func (s *Server) saveState() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.SaveState(s.state)
}

func (s *Server) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shuttingDown
}

func (s *Server) signalWork() {
	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
}

func (s *Server) checkDedup(routeID, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneDedupLocked()
	_, ok := s.dedup[routeID+"|"+key]
	return ok
}

func (s *Server) markDedup(routeID, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dedup[routeID+"|"+key] = s.now().Add(time.Duration(s.cfg.DedupTTLSeconds) * time.Second)
}

func (s *Server) pruneDedupLocked() {
	now := s.now()
	for key, expiresAt := range s.dedup {
		if now.After(expiresAt) {
			delete(s.dedup, key)
		}
	}
}

func (s *Server) pendingCountForRouteLocked(routeID string) int {
	count := 0
	for _, event := range s.pending {
		if event.route.ID == routeID {
			count++
		}
	}
	return count
}

func (s *Server) runtimeIdle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending) == 0 && len(s.routeRunning) == 0
}

func (s *Server) rejectPending(reason string, status int) {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()

	for _, event := range pending {
		event.run.Outcome = store.RunRejected
		event.run.FailureReason = reason
		event.run.HTTPStatus = status
		event.run.FinishedAt = s.now().UTC().Format(time.RFC3339)
		_ = s.store.AppendRun(event.run)
	}
}

func resolveBinary(name, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return exec.LookPath(name)
}

func writeResponse(w io.Writer, resp Response) error {
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(append(raw, '\n'))
	return err
}

func decodePayload(raw json.RawMessage, dst any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(raw, '\n'))
}

func statusFromBool(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateLoopbackWebhookAddr(addr string) error {
	return store.ValidateWebhookHTTPAddr(addr)
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("admin socket path exists and is not a socket: %s", path)
	}
	return os.Remove(path)
}

func (s *Server) appendWebhookRouteRejection(route store.RouteRecord, status int, reason, dedupKey string) {
	s.appendWebhookRejection(route.ID, route.WorkspaceID, route.EpicSlug, status, reason, dedupKey)
}

func (s *Server) appendWebhookRejection(routeID, workspaceID, epicSlug string, status int, reason, dedupKey string) {
	runID, err := store.GenerateID("run_")
	if err != nil {
		return
	}
	_ = s.store.AppendRun(store.RunRecord{
		ID:            runID,
		RouteID:       routeID,
		WorkspaceID:   workspaceID,
		EpicSlug:      epicSlug,
		TriggerType:   store.RouteTypeWebhook,
		DedupKey:      dedupKey,
		Outcome:       store.RunRejected,
		FailureReason: reason,
		EnqueuedAt:    s.now().UTC().Format(time.RFC3339),
		FinishedAt:    s.now().UTC().Format(time.RFC3339),
		HTTPStatus:    status,
	})
}
