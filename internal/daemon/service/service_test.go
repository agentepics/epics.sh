package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderLaunchdPlist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	manager := NewManager(filepath.Join(home, ".config", "epicsd"), "/usr/local/bin/epicsd")
	manager.GOOS = "darwin"

	path, content, err := manager.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasSuffix(path, "Library/LaunchAgents/com.agentepics.epicsd.plist") {
		t.Fatalf("unexpected path: %s", path)
	}
	if !strings.Contains(content, "<string>/usr/local/bin/epicsd</string>") {
		t.Fatalf("missing binary path: %s", content)
	}
	if !strings.Contains(content, "EPICSD_HOME") {
		t.Fatalf("missing daemon home env: %s", content)
	}
}

func TestLinuxControlCommandsAndUnitWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var commands []string
	manager := NewManager(filepath.Join(home, ".config", "epicsd"), "/usr/local/bin/epicsd")
	manager.GOOS = "linux"
	manager.Run = func(ctx context.Context, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}

	if err := manager.Install(context.Background()); err != nil {
		t.Fatalf("install: %v", err)
	}
	unitPath := filepath.Join(home, ".config", "systemd", "user", "epicsd.service")
	raw, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read unit: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "ExecStart=/usr/local/bin/epicsd") {
		t.Fatalf("unexpected unit content: %s", content)
	}
	if len(commands) != 2 || commands[0] != "systemctl --user daemon-reload" || commands[1] != "systemctl --user enable --now epicsd.service" {
		t.Fatalf("unexpected install commands: %+v", commands)
	}

	commands = nil
	if err := manager.Restart(context.Background()); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if len(commands) != 1 || commands[0] != "systemctl --user restart epicsd.service" {
		t.Fatalf("unexpected restart commands: %+v", commands)
	}
}
