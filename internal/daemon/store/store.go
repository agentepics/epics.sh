package store

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type workspaceList struct {
	Items []WorkspaceRecord `json:"items"`
}

type routeList struct {
	Items []RouteRecord `json:"items"`
}

type Store struct {
	Home string
	now  func() time.Time
}

const maxRunOutputBytes = 32 << 10

func ResolveHome() (string, error) {
	if override := strings.TrimSpace(os.Getenv("EPICSD_HOME")); override != "" {
		return filepath.Clean(override), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "epicsd"), nil
}

func MustResolveHome() string {
	home, err := ResolveHome()
	if err != nil {
		panic(err)
	}
	return home
}

func DefaultConfig(home string) Config {
	return Config{
		AdminSocketPath:         filepath.Join(home, "epicsd.sock"),
		WebhookHTTPAddr:         "127.0.0.1:42617",
		MaxBodyBytes:            1 << 20,
		GlobalQueueCapacity:     256,
		PerWorkspaceConcurrency: 4,
		DedupTTLSeconds:         300,
		SchedulerTickSeconds:    30,
		AllowInsecureAuthNone:   false,
		ShutdownTimeoutSeconds:  30,
	}
}

func Open(home string) *Store {
	return &Store{
		Home: filepath.Clean(home),
		now:  time.Now,
	}
}

func (s *Store) ConfigPath() string     { return filepath.Join(s.Home, "config.json") }
func (s *Store) StatePath() string      { return filepath.Join(s.Home, "state.json") }
func (s *Store) WorkspacesPath() string { return filepath.Join(s.Home, "workspaces.json") }
func (s *Store) RoutesPath() string     { return filepath.Join(s.Home, "routes.json") }
func (s *Store) RunsDir() string        { return filepath.Join(s.Home, "runs") }
func (s *Store) RunOutputsDir() string  { return filepath.Join(s.RunsDir(), "output") }
func (s *Store) SecretsDir() string     { return filepath.Join(s.Home, "secrets") }
func (s *Store) LogPath() string        { return filepath.Join(s.Home, "epicsd.log") }
func (s *Store) SocketPath() string     { return filepath.Join(s.Home, "epicsd.sock") }

func (s *Store) Ensure() error {
	if err := os.MkdirAll(s.Home, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.RunsDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.RunOutputsDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.SecretsDir(), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(s.ConfigPath()); errors.Is(err, os.ErrNotExist) {
		if err := s.SaveConfig(DefaultConfig(s.Home)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(s.WorkspacesPath()); errors.Is(err, os.ErrNotExist) {
		if err := s.SaveWorkspaces(nil); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(s.RoutesPath()); errors.Is(err, os.ErrNotExist) {
		if err := s.SaveRoutes(nil); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(s.StatePath()); errors.Is(err, os.ErrNotExist) {
		if err := s.SaveState(State{Status: HealthOK, DegradedSubsystems: []string{}}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Store) LoadConfig() (Config, error) {
	if err := s.Ensure(); err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := readJSONFile(s.ConfigPath(), &cfg); err != nil {
		return Config{}, err
	}
	return normalizeConfig(s.Home, cfg)
}

func (s *Store) SaveConfig(cfg Config) error {
	normalized, err := normalizeConfig(s.Home, cfg)
	if err != nil {
		return err
	}
	return writeJSONFileAtomically(s.ConfigPath(), normalized, 0o644)
}

func (s *Store) LoadState() (State, error) {
	if err := s.Ensure(); err != nil {
		return State{}, err
	}
	var state State
	if err := readJSONFile(s.StatePath(), &state); err != nil {
		return State{}, err
	}
	if state.Status == "" {
		state.Status = HealthOK
	}
	if state.DegradedSubsystems == nil {
		state.DegradedSubsystems = []string{}
	}
	return state, nil
}

func (s *Store) SaveState(state State) error {
	if state.Status == "" {
		state.Status = HealthOK
	}
	if state.DegradedSubsystems == nil {
		state.DegradedSubsystems = []string{}
	}
	return writeJSONFileAtomically(s.StatePath(), state, 0o644)
}

func (s *Store) LoadWorkspaces() ([]WorkspaceRecord, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	var payload workspaceList
	if err := readJSONFile(s.WorkspacesPath(), &payload); err != nil {
		return nil, err
	}
	if payload.Items == nil {
		payload.Items = []WorkspaceRecord{}
	}
	sort.Slice(payload.Items, func(i, j int) bool {
		return payload.Items[i].ID < payload.Items[j].ID
	})
	return payload.Items, nil
}

func (s *Store) SaveWorkspaces(items []WorkspaceRecord) error {
	if items == nil {
		items = []WorkspaceRecord{}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return writeJSONFileAtomically(s.WorkspacesPath(), workspaceList{Items: items}, 0o644)
}

func (s *Store) LoadRoutes() ([]RouteRecord, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	var payload routeList
	if err := readJSONFile(s.RoutesPath(), &payload); err != nil {
		return nil, err
	}
	if payload.Items == nil {
		payload.Items = []RouteRecord{}
	}
	sort.Slice(payload.Items, func(i, j int) bool {
		return payload.Items[i].ID < payload.Items[j].ID
	})
	return payload.Items, nil
}

func (s *Store) SaveRoutes(items []RouteRecord) error {
	if items == nil {
		items = []RouteRecord{}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return writeJSONFileAtomically(s.RoutesPath(), routeList{Items: items}, 0o644)
}

func (s *Store) WriteSecret(routeID, mode, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("secret value cannot be empty")
	}
	if err := os.MkdirAll(s.SecretsDir(), 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(routeID + "|" + mode))
	ref := hex.EncodeToString(sum[:]) + ".secret"
	path := filepath.Join(s.SecretsDir(), ref)
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return ref, nil
}

func (s *Store) ReadSecret(ref string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(s.SecretsDir(), filepath.Base(ref)))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Store) RemoveSecret(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return nil
	}
	err := os.Remove(filepath.Join(s.SecretsDir(), filepath.Base(ref)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Store) AppendRun(run RunRecord) error {
	if err := os.MkdirAll(s.RunsDir(), 0o755); err != nil {
		return err
	}
	stamp := run.EnqueuedAt
	if strings.TrimSpace(stamp) == "" {
		stamp = s.now().UTC().Format(time.RFC3339)
	}
	ts, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		ts = s.now().UTC()
	}
	path := filepath.Join(s.RunsDir(), ts.UTC().Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	_, err = file.Write(raw)
	return err
}

func (s *Store) ListRuns(routeID, workspaceID string, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	files, err := s.runFilesNewestFirst()
	if err != nil {
		return nil, err
	}
	var runs []RunRecord
	for _, path := range files {
		records, err := readRunFile(path)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if routeID != "" && record.RouteID != routeID {
				continue
			}
			if workspaceID != "" && record.WorkspaceID != workspaceID {
				continue
			}
			runs = append(runs, record)
		}
	}
	sort.Slice(runs, func(i, j int) bool {
		return runSortKey(runs[i]).After(runSortKey(runs[j]))
	})
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (s *Store) InspectRun(id string) (RunRecord, bool, error) {
	files, err := s.runFilesNewestFirst()
	if err != nil {
		return RunRecord{}, false, err
	}
	for _, path := range files {
		records, err := readRunFile(path)
		if err != nil {
			return RunRecord{}, false, err
		}
		for _, record := range records {
			if record.ID == id {
				return record, true, nil
			}
		}
	}
	return RunRecord{}, false, nil
}

func (s *Store) WriteRunOutput(runID string, output string) (string, error) {
	if strings.TrimSpace(runID) == "" {
		return "", errors.New("run id is required")
	}
	if err := os.MkdirAll(s.RunOutputsDir(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(s.RunOutputsDir(), filepath.Base(runID)+".log")
	trimmed := trimRunOutput(output)
	if trimmed == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return "", nil
	}
	if !strings.HasSuffix(trimmed, "\n") {
		trimmed += "\n"
	}
	if err := writeFileAtomically(path, []byte(trimmed), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ReadRunOutput(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimRight(string(raw), "\n"), nil
}

func GenerateID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}

func trimRunOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	if len(output) <= maxRunOutputBytes {
		return output
	}
	marker := "\n[truncated]\n"
	limit := maxRunOutputBytes - len(marker)
	if limit < 0 {
		limit = 0
	}
	return output[:limit] + marker
}

func readJSONFile(path string, dst any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func writeJSONFileAtomically(path string, payload any, mode os.FileMode) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeFileAtomically(path, raw, mode)
}

func writeFileAtomically(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(contents); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err == nil {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempPath, path)
}

func (s *Store) runFilesNewestFirst() ([]string, error) {
	if err := os.MkdirAll(s.RunsDir(), 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.RunsDir())
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		files = append(files, filepath.Join(s.RunsDir(), entry.Name()))
	}
	sort.Slice(files, func(i, j int) bool {
		return filepath.Base(files[i]) > filepath.Base(files[j])
	})
	return files, nil
}

func readRunFile(path string) ([]RunRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []RunRecord
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var record RunRecord
			if decodeErr := json.Unmarshal(bytesTrimSpace(line), &record); decodeErr != nil {
				return nil, fmt.Errorf("decode %s: %w", path, decodeErr)
			}
			records = append(records, record)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

func runSortKey(run RunRecord) time.Time {
	for _, value := range []string{run.FinishedAt, run.StartedAt, run.EnqueuedAt} {
		if ts, err := time.Parse(time.RFC3339, value); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func normalizeConfig(home string, cfg Config) (Config, error) {
	defaults := DefaultConfig(home)
	if cfg.AdminSocketPath == "" {
		cfg.AdminSocketPath = defaults.AdminSocketPath
	}
	if cfg.WebhookHTTPAddr == "" {
		cfg.WebhookHTTPAddr = defaults.WebhookHTTPAddr
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = defaults.MaxBodyBytes
	}
	if cfg.GlobalQueueCapacity == 0 {
		cfg.GlobalQueueCapacity = defaults.GlobalQueueCapacity
	}
	if cfg.PerWorkspaceConcurrency == 0 {
		cfg.PerWorkspaceConcurrency = defaults.PerWorkspaceConcurrency
	}
	if cfg.DedupTTLSeconds == 0 {
		cfg.DedupTTLSeconds = defaults.DedupTTLSeconds
	}
	if cfg.SchedulerTickSeconds == 0 {
		cfg.SchedulerTickSeconds = defaults.SchedulerTickSeconds
	}
	if cfg.ShutdownTimeoutSeconds == 0 {
		cfg.ShutdownTimeoutSeconds = defaults.ShutdownTimeoutSeconds
	}
	if err := ValidateWebhookHTTPAddr(cfg.WebhookHTTPAddr); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ValidateWebhookHTTPAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid webhook_http_addr %q: %w", addr, err)
	}
	if host != "127.0.0.1" {
		return fmt.Errorf("webhook_http_addr must bind to 127.0.0.1, got %q", host)
	}
	if strings.TrimSpace(port) == "" {
		return fmt.Errorf("invalid webhook_http_addr %q: missing port", addr)
	}
	return nil
}
