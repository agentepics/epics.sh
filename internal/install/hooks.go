package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agentepics/epics.sh/internal/epic"
)

type installHookRecord struct {
	Trigger        string   `json:"trigger"`
	EpicID         string   `json:"epicId"`
	Timestamp      string   `json:"timestamp"`
	IdempotencyKey string   `json:"idempotencyKey"`
	Handlers       []string `json:"handlers"`
	CurrentPlan    string   `json:"currentPlan,omitempty"`
}

func RunInstallHooks(pkg epic.Package) error {
	handlers, err := discoverInstallHookHandlers(pkg.Root)
	if err != nil {
		return err
	}
	if len(handlers) == 0 {
		return nil
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	idempotencyKey, err := generateIdempotencyKey()
	if err != nil {
		return err
	}

	state, _, err := epic.ReadState(pkg)
	if err != nil {
		return err
	}
	currentPlan := currentPlanPath(pkg, state)
	payload := map[string]any{
		"trigger":         "install",
		"epic_id":         pkg.EpicID,
		"timestamp":       timestamp,
		"idempotency_key": idempotencyKey,
	}
	if state != nil {
		payload["state"] = state
	}
	if currentPlan != "" {
		payload["current_plan"] = currentPlan
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var recordedHandlers []string
	for _, handler := range handlers {
		rel := epic.RelativePath(pkg.Root, handler)
		hookType, err := installHookType(handler)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		switch hookType {
		case "script":
			if err := runScriptInstallHook(pkg.Root, handler, rawPayload); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
		case "prompt":
			if err := runPromptInstallHook(pkg.Root, handler, rawPayload); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
		default:
			return fmt.Errorf("%s uses unsupported install hook type %q", rel, hookType)
		}
		recordedHandlers = append(recordedHandlers, rel)
	}

	record := installHookRecord{
		Trigger:        "install",
		EpicID:         pkg.EpicID,
		Timestamp:      timestamp,
		IdempotencyKey: idempotencyKey,
		Handlers:       recordedHandlers,
		CurrentPlan:    currentPlan,
	}
	return writeInstallHookRecord(pkg.Root, record)
}

func discoverInstallHookHandlers(root string) ([]string, error) {
	dirPath := filepath.Join(root, "hooks", "install.d")
	if info, err := os.Stat(dirPath); err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("hooks/install.d must be a directory when present")
		}
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		var handlers []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			handlers = append(handlers, filepath.Join(dirPath, entry.Name()))
		}
		sort.Strings(handlers)
		return handlers, nil
	}

	matches, err := filepath.Glob(filepath.Join(root, "hooks", "install.*"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func installHookType(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md":
		return "prompt", nil
	case ".yml", ".yaml":
		return "http", nil
	case ".sh", ".bash", ".zsh", ".py":
		return "script", nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&0o111 != 0 {
		return "script", nil
	}
	return "", errors.New("install hook is not a supported script file")
}

func runScriptInstallHook(root, path string, payload []byte) error {
	cmd, err := scriptCommand(path)
	if err != nil {
		return err
	}
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install hook failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func runPromptInstallHook(root, path string, payload []byte) error {
	body := markdownBody(path)
	if strings.TrimSpace(body) == "" {
		return errors.New("prompt install hook body is empty")
	}
	if _, ok := os.LookupEnv("ANTHROPIC_API_KEY"); !ok {
		return errors.New("prompt install hook requires ANTHROPIC_API_KEY")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.New("prompt install hook requires the claude CLI")
	}

	prompt := strings.TrimSpace(
		"You are executing an EPIC install hook in the current working directory.\n" +
			"The current directory is the installed Epic root.\n" +
			"Use the event context below and follow the hook instructions exactly.\n" +
			"Make the required file changes in the workspace, then respond with a short plain-text confirmation.\n\n" +
			"Event context JSON:\n" + string(payload) + "\n\n" +
			"Hook instructions:\n" + body,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--dangerously-skip-permissions", "--output-format", "text")
	cmd.Dir = root
	cmd.Env = claudeCommandEnv(root)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New("prompt install hook timed out")
	}
	if err != nil {
		return fmt.Errorf("prompt install hook failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func claudeCommandEnv(root string) []string {
	env := os.Environ()
	if os.Getenv("HOME") != "" {
		return env
	}
	home := filepath.Join(root, ".claude-home")
	_ = os.MkdirAll(home, 0o755)
	return append(env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
	)
}

func scriptCommand(path string) (*exec.Cmd, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh", ".bash", ".zsh":
		return exec.Command("sh", path), nil
	case ".py":
		if _, err := exec.LookPath("python3"); err != nil {
			return nil, errors.New("python3 is required for Python install hooks")
		}
		return exec.Command("python3", path), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&0o111 == 0 {
		return nil, errors.New("script hook must be executable")
	}
	return exec.Command(path), nil
}

func writeInstallHookRecord(root string, record installHookRecord) error {
	path := filepath.Join(root, "runtime", "install.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func markdownBody(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimSpace(content)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		}
	}
	return strings.TrimSpace(content)
}

func currentPlanPath(pkg epic.Package, state map[string]any) string {
	if value := epic.LookupString(state, "current_plan", "currentPlan"); value != "" {
		return filepath.ToSlash(value)
	}
	if len(pkg.PlanFiles) == 0 {
		return ""
	}
	return epic.RelativePath(pkg.Root, pkg.PlanFiles[len(pkg.PlanFiles)-1])
}

func generateIdempotencyKey() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
