package hostutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/agentepics/epics.sh/internal/doctor"
	"github.com/agentepics/epics.sh/internal/hostapi"
)

type WriteState int

const (
	WriteCreated WriteState = iota
	WriteUnchanged
	WriteSkipped
)

func WriteIfMissingOrSame(path, content string) (WriteState, error) {
	if raw, err := os.ReadFile(path); err == nil {
		if strings.TrimSpace(string(raw)) == strings.TrimSpace(content) {
			return WriteUnchanged, nil
		}
		// Additive-only: never overwrite existing host files with different content.
		return WriteSkipped, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return WriteCreated, err
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return WriteCreated, err
	}
	return WriteCreated, nil
}

func AppendSection(path, section string) (WriteState, error) {
	trimmedSection := strings.TrimSpace(section)
	raw, err := os.ReadFile(path)
	if err == nil {
		content := strings.TrimSpace(string(raw))
		if strings.Contains(content, trimmedSection) {
			return WriteUnchanged, nil
		}
		if content == "" {
			content = trimmedSection
		} else {
			content += "\n\n" + trimmedSection
		}
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			return WriteSkipped, err
		}
		return WriteCreated, nil
	}
	if !os.IsNotExist(err) {
		return WriteSkipped, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return WriteCreated, err
	}
	if err := os.WriteFile(path, []byte(trimmedSection+"\n"), 0o644); err != nil {
		return WriteCreated, err
	}
	return WriteCreated, nil
}

func RecordWrite(result *hostapi.Result, path string, state WriteState) {
	path = filepath.ToSlash(path)
	switch state {
	case WriteCreated:
		result.Created = append(result.Created, path)
	case WriteUnchanged:
		result.Unchanged = append(result.Unchanged, path)
	case WriteSkipped:
		result.Skipped = append(result.Skipped, path)
	}
}

func HostDoctorChecks(cwd, hostName, managedDir, instructionFile, instructionMarker string) ([]doctor.Check, error) {
	commandDir := filepath.Join(cwd, managedDir, "commands")
	skillsDir := filepath.Join(cwd, managedDir, "skills")
	checks := []doctor.Check{
		CheckHostManagedDir(hostName, filepath.Join(cwd, managedDir)),
		CheckHostCommands(hostName, commandDir),
		CheckHostInstructionFile(hostName, filepath.Join(cwd, instructionFile), instructionMarker),
		CheckHostSkills(hostName, skillsDir),
	}
	return checks, nil
}

func CheckHostManagedDir(hostName, dir string) doctor.Check {
	name := hostName + "-managed-dir"
	if info, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return doctor.Check{Name: name, Status: "fail", Message: filepath.ToSlash(dir) + " does not exist"}
		}
		return doctor.Check{Name: name, Status: "fail", Message: err.Error()}
	} else if !info.IsDir() {
		return doctor.Check{Name: name, Status: "fail", Message: filepath.ToSlash(dir) + " is not a directory"}
	}

	tempFile := filepath.Join(dir, ".epics-host-doctor.tmp")
	if err := os.WriteFile(tempFile, []byte("ok\n"), 0o644); err != nil {
		return doctor.Check{Name: name, Status: "fail", Message: err.Error()}
	}
	_ = os.Remove(tempFile)
	return doctor.Check{Name: name, Status: "ok", Message: filepath.ToSlash(dir) + " exists and is writable"}
}

func CheckHostCommands(hostName, dir string) doctor.Check {
	name := hostName + "-commands"
	required := []string{"epics-resume.md", "epics-info.md", "epics-doctor.md"}
	var missing []string
	var empty []string
	for _, file := range required {
		path := filepath.Join(dir, file)
		raw, err := os.ReadFile(path)
		if err != nil {
			missing = append(missing, filepath.ToSlash(path))
			continue
		}
		if strings.TrimSpace(string(raw)) == "" {
			empty = append(empty, filepath.ToSlash(path))
		}
	}
	if len(missing) > 0 || len(empty) > 0 {
		parts := []string{}
		if len(missing) > 0 {
			parts = append(parts, "missing: "+strings.Join(missing, ", "))
		}
		if len(empty) > 0 {
			parts = append(parts, "empty: "+strings.Join(empty, ", "))
		}
		return doctor.Check{Name: name, Status: "fail", Message: strings.Join(parts, "; ")}
	}
	return doctor.Check{Name: name, Status: "ok", Message: filepath.ToSlash(dir) + " contains non-empty Epic command files"}
}

func CheckHostInstructionFile(hostName, path, marker string) doctor.Check {
	name := hostName + "-instructions"
	raw, err := os.ReadFile(path)
	if err != nil {
		return doctor.Check{Name: name, Status: "fail", Message: err.Error()}
	}
	if !strings.Contains(string(raw), marker) {
		return doctor.Check{Name: name, Status: "fail", Message: filepath.ToSlash(path) + " is missing Epic guidance"}
	}
	return doctor.Check{Name: name, Status: "ok", Message: filepath.ToSlash(path) + " contains Epic guidance"}
}

func CheckHostSkills(hostName, dir string) doctor.Check {
	name := hostName + "-skills"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return doctor.Check{Name: name, Status: "fail", Message: filepath.ToSlash(dir) + " does not exist"}
		}
		return doctor.Check{Name: name, Status: "fail", Message: err.Error()}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		if raw, err := os.ReadFile(skillPath); err == nil && strings.TrimSpace(string(raw)) != "" {
			return doctor.Check{Name: name, Status: "ok", Message: "found installed Epic at " + filepath.ToSlash(filepath.Join(dir, entry.Name()))}
		}
	}
	return doctor.Check{Name: name, Status: "fail", Message: "no installed Epics found in " + filepath.ToSlash(dir)}
}
