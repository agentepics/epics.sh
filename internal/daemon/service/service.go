package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	launchdLabel = "com.agentepics.epicsd"
	systemdUnit  = "epicsd.service"
)

type Runner func(ctx context.Context, name string, args ...string) error

type Manager struct {
	DaemonHome string
	BinaryPath string
	GOOS       string
	Run        Runner
}

func NewManager(home, binaryPath string) *Manager {
	return &Manager{
		DaemonHome: home,
		BinaryPath: binaryPath,
		GOOS:       runtime.GOOS,
		Run: func(ctx context.Context, name string, args ...string) error {
			cmd := exec.CommandContext(ctx, name, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				text := strings.TrimSpace(string(output))
				if text == "" {
					return err
				}
				return fmt.Errorf("%w: %s", err, text)
			}
			return nil
		},
	}
}

func ResolveBinary() (string, error) {
	if override := strings.TrimSpace(os.Getenv("EPICSD_BIN")); override != "" {
		return filepath.Clean(override), nil
	}
	if path, err := exec.LookPath("epicsd"); err == nil {
		return path, nil
	}
	current, err := os.Executable()
	if err != nil {
		return "", errors.New("could not resolve epicsd binary; set EPICSD_BIN or put epicsd on PATH")
	}
	sibling := filepath.Join(filepath.Dir(current), "epicsd")
	if _, err := os.Stat(sibling); err == nil {
		return sibling, nil
	}
	return "", errors.New("could not resolve epicsd binary; set EPICSD_BIN or put epicsd on PATH")
}

func (m *Manager) Install(ctx context.Context) error {
	path, content, err := m.Render()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	switch m.GOOS {
	case "darwin":
		target := launchdDomain()
		if err := m.Run(ctx, "launchctl", "enable", target+"/"+launchdLabel); err != nil {
			return err
		}
		if err := m.Run(ctx, "launchctl", "bootstrap", target, path); err != nil {
			return err
		}
		return m.Run(ctx, "launchctl", "kickstart", "-k", target+"/"+launchdLabel)
	case "linux":
		if err := m.Run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
		return m.Run(ctx, "systemctl", "--user", "enable", "--now", systemdUnit)
	default:
		return fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
}

func (m *Manager) Uninstall(ctx context.Context) error {
	path, _, err := m.Render()
	if err != nil {
		return err
	}
	switch m.GOOS {
	case "darwin":
		target := launchdDomain()
		_ = m.Run(ctx, "launchctl", "bootout", target+"/"+launchdLabel)
		_ = m.Run(ctx, "launchctl", "disable", target+"/"+launchdLabel)
	case "linux":
		_ = m.Run(ctx, "systemctl", "--user", "disable", "--now", systemdUnit)
		_ = m.Run(ctx, "systemctl", "--user", "daemon-reload")
	default:
		return fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if m.GOOS == "linux" {
		_ = m.Run(ctx, "systemctl", "--user", "daemon-reload")
	}
	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	path, _, err := m.Render()
	if err != nil {
		return err
	}
	switch m.GOOS {
	case "darwin":
		return m.Run(ctx, "launchctl", "bootstrap", launchdDomain(), path)
	case "linux":
		return m.Run(ctx, "systemctl", "--user", "start", systemdUnit)
	default:
		return fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	switch m.GOOS {
	case "darwin":
		return m.Run(ctx, "launchctl", "bootout", launchdDomain()+"/"+launchdLabel)
	case "linux":
		return m.Run(ctx, "systemctl", "--user", "stop", systemdUnit)
	default:
		return fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
}

func (m *Manager) Restart(ctx context.Context) error {
	path, _, err := m.Render()
	if err != nil {
		return err
	}
	switch m.GOOS {
	case "darwin":
		target := launchdDomain()
		_ = m.Run(ctx, "launchctl", "bootout", target+"/"+launchdLabel)
		if err := m.Run(ctx, "launchctl", "bootstrap", target, path); err != nil {
			return err
		}
		return m.Run(ctx, "launchctl", "kickstart", "-k", target+"/"+launchdLabel)
	case "linux":
		return m.Run(ctx, "systemctl", "--user", "restart", systemdUnit)
	default:
		return fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
}

func (m *Manager) Render() (string, string, error) {
	if strings.TrimSpace(m.BinaryPath) == "" {
		return "", "", errors.New("epicsd binary path is required")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	switch m.GOOS {
	case "darwin":
		path := filepath.Join(homeDir, "Library", "LaunchAgents", launchdLabel+".plist")
		content := renderLaunchdPlist(m.BinaryPath, m.DaemonHome)
		return path, content, nil
	case "linux":
		path := filepath.Join(homeDir, ".config", "systemd", "user", systemdUnit)
		content := renderSystemdUnit(m.BinaryPath, m.DaemonHome)
		return path, content, nil
	default:
		return "", "", fmt.Errorf("unsupported service manager on %s", m.GOOS)
	}
}

func renderLaunchdPlist(binaryPath, daemonHome string) string {
	logPath := filepath.Join(daemonHome, "epicsd.log")
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>` + launchdLabel + `</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + xmlEscape(binaryPath) + `</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>EnvironmentVariables</key>
  <dict>
    <key>EPICSD_HOME</key>
    <string>` + xmlEscape(daemonHome) + `</string>
  </dict>
  <key>StandardOutPath</key>
  <string>` + xmlEscape(logPath) + `</string>
  <key>StandardErrorPath</key>
  <string>` + xmlEscape(logPath) + `</string>
</dict>
</plist>
`
}

func renderSystemdUnit(binaryPath, daemonHome string) string {
	logPath := filepath.Join(daemonHome, "epicsd.log")
	return `[Unit]
Description=epicsd user daemon
After=default.target

[Service]
Type=simple
Environment=EPICSD_HOME=` + shellEscape(daemonHome) + `
ExecStart=` + shellEscape(binaryPath) + `
Restart=on-failure
StandardOutput=append:` + shellEscape(logPath) + `
StandardError=append:` + shellEscape(logPath) + `

[Install]
WantedBy=default.target
`
}

func launchdDomain() string {
	return "gui/" + strconv.Itoa(os.Getuid())
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func shellEscape(value string) string {
	return strings.ReplaceAll(value, " ", `\ `)
}
