package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/workspace"
)

const (
	stateFileName = "state.json"
	stateCorePath = "state/core.json"
	stateLockName = ".state.lock"
)

type Snapshot struct {
	Path string         `json:"path"`
	Data map[string]any `json:"data"`
}

func ResolvePath(cwd string) (string, bool, error) {
	specVersion := detectSpecVersion(cwd)
	corePath := epic.RuntimePath(cwd, specVersion, stateCorePath)
	if _, err := os.Stat(corePath); err == nil {
		return corePath, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	statePath := epic.RuntimePath(cwd, specVersion, stateFileName)
	if _, err := os.Stat(statePath); err == nil {
		return statePath, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	return statePath, false, nil
}

func detectSpecVersion(cwd string) string {
	pkg, err := epic.Load(cwd)
	if err != nil {
		return ""
	}
	return pkg.SpecVersion
}

func Read(cwd string) (Snapshot, error) {
	path, exists, err := ResolvePath(cwd)
	if err != nil {
		return Snapshot{}, err
	}
	if !exists {
		return Snapshot{Path: path, Data: map[string]any{}}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Snapshot{Path: path, Data: map[string]any{}}, nil
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return Snapshot{}, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return Snapshot{Path: path, Data: data}, nil
}

func Get(cwd, key string) (any, Snapshot, error) {
	snapshot, err := Read(cwd)
	if err != nil {
		return nil, Snapshot{}, err
	}
	if strings.TrimSpace(key) == "" {
		return snapshot.Data, snapshot, nil
	}

	value, ok := lookup(snapshot.Data, key)
	if !ok {
		return nil, snapshot, fmt.Errorf("state key %q not found", key)
	}
	return value, snapshot, nil
}

func Set(cwd, key, rawValue string) (Snapshot, any, error) {
	parts, err := splitKey(key)
	if err != nil {
		return Snapshot{}, nil, err
	}

	value := parseValue(rawValue)

	release, err := acquireLock(cwd)
	if err != nil {
		return Snapshot{}, nil, err
	}
	defer release()

	current, err := Read(cwd)
	if err != nil {
		return Snapshot{}, nil, err
	}
	if current.Data == nil {
		current.Data = map[string]any{}
	}
	assign(current.Data, parts, value)

	raw, err := json.MarshalIndent(current.Data, "", "  ")
	if err != nil {
		return Snapshot{}, nil, err
	}
	raw = append(raw, '\n')
	if err := writeFileAtomically(current.Path, raw, 0o644); err != nil {
		return Snapshot{}, nil, err
	}

	return Snapshot{Path: current.Path, Data: current.Data}, value, nil
}

func lookup(data map[string]any, key string) (any, bool) {
	parts, err := splitKey(key)
	if err != nil {
		return nil, false
	}

	var current any = data
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok := object[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func assign(data map[string]any, parts []string, value any) {
	current := data
	for _, part := range parts[:len(parts)-1] {
		nested, ok := current[part].(map[string]any)
		if !ok {
			nested = map[string]any{}
			current[part] = nested
		}
		current = nested
	}
	current[parts[len(parts)-1]] = value
}

func parseValue(raw string) any {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		return value
	}
	return raw
}

func splitKey(key string) ([]string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("state key must not be empty")
	}

	parts := strings.Split(key, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("invalid state key %q", key)
		}
	}
	return parts, nil
}

func acquireLock(cwd string) (func(), error) {
	if err := workspace.EnsureManagedDir(cwd); err != nil {
		return nil, err
	}

	lockPath := filepath.Join(workspace.ManagedDir(cwd), stateLockName)
	deadline := time.Now().Add(5 * time.Second)

	for {
		err := os.Mkdir(lockPath, 0o755)
		if err == nil {
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for state lock %s", filepath.ToSlash(lockPath))
		}
		time.Sleep(25 * time.Millisecond)
	}
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
	return replaceFile(tempPath, path, mode, os.Rename)
}

func replaceFile(tempPath, path string, mode os.FileMode, rename func(string, string) error) error {
	if err := rename(tempPath, path); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	src, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Chmod(mode); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return nil
}
