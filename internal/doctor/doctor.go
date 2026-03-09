package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/workspace"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Result struct {
	Checks []Check `json:"checks"`
}

func Run(cwd string) (Result, error) {
	installs, err := workspace.LoadInstalls(cwd)
	if err != nil {
		return Result{}, err
	}

	checks := []Check{
		checkManagedDir(cwd),
		checkWorkspaceWritable(cwd),
		checkAuthoredPackage(cwd, installs),
		checkInstalledEpics(cwd, installs),
		checkInstallSources(cwd, installs),
	}

	return Result{Checks: checks}, nil
}

func HasFailures(result Result) bool {
	for _, check := range result.Checks {
		if check.Status == "fail" {
			return true
		}
	}
	return false
}

func checkManagedDir(cwd string) Check {
	if err := workspace.EnsureManagedDir(cwd); err != nil {
		return Check{Name: "managed-dir", Status: "fail", Message: err.Error()}
	}

	tempFile := filepath.Join(workspace.ManagedDir(cwd), ".doctor.tmp")
	if err := os.WriteFile(tempFile, []byte("ok\n"), 0o644); err != nil {
		return Check{Name: "managed-dir", Status: "fail", Message: err.Error()}
	}
	_ = os.Remove(tempFile)
	return Check{Name: "managed-dir", Status: "ok", Message: "workspace-managed install directory is writable"}
}

func checkWorkspaceWritable(cwd string) Check {
	tempFile := filepath.Join(cwd, ".epics-doctor.tmp")
	if err := os.WriteFile(tempFile, []byte("ok\n"), 0o644); err != nil {
		return Check{Name: "workspace-write", Status: "fail", Message: err.Error()}
	}
	_ = os.Remove(tempFile)
	return Check{Name: "workspace-write", Status: "ok", Message: "current workspace is writable"}
}

func checkAuthoredPackage(cwd string, installs []workspace.InstallRecord) Check {
	pkg, diagnostics, err := epic.Validate(cwd)
	if err == nil && (pkg.SkillPath != "" || pkg.EpicPath != "") {
		status := "ok"
		message := "current directory contains a valid authored Epic package"
		if epic.HasErrors(diagnostics) {
			status = "fail"
			message = "current directory contains an invalid authored Epic package"
		}
		return Check{Name: "authored-package", Status: status, Message: message}
	}

	if len(installs) > 0 {
		return Check{
			Name:    "authored-package",
			Status:  "ok",
			Message: fmt.Sprintf("no authored Epic package in the current directory; workspace tracks %d installed Epic(s)", len(installs)),
		}
	}

	return Check{Name: "authored-package", Status: "ok", Message: "no authored Epic package in the current directory"}
}

func checkInstalledEpics(cwd string, installs []workspace.InstallRecord) Check {
	if len(installs) == 0 {
		return Check{Name: "installed-epics", Status: "ok", Message: "no installed Epics recorded in workspace metadata"}
	}

	var missing []string
	var labels []string
	for _, install := range installs {
		labels = append(labels, fmt.Sprintf("%s@%s", install.Slug, install.Host))
		installPath := filepath.Join(cwd, filepath.FromSlash(install.InstalledDir))
		if _, err := os.Stat(installPath); err != nil {
			missing = append(missing, filepath.ToSlash(install.InstalledDir))
		}
	}
	sort.Strings(labels)
	if len(missing) > 0 {
		sort.Strings(missing)
		return Check{
			Name:    "installed-epics",
			Status:  "warning",
			Message: fmt.Sprintf("workspace metadata tracks %d installed Epic(s), but some install paths are missing: %s", len(installs), strings.Join(missing, ", ")),
		}
	}

	return Check{
		Name:    "installed-epics",
		Status:  "ok",
		Message: fmt.Sprintf("workspace metadata tracks %d installed Epic(s): %s", len(installs), strings.Join(labels, ", ")),
	}
}

func checkInstallSources(cwd string, installs []workspace.InstallRecord) Check {
	localInstalls := filterLocalInstalls(installs)
	if len(localInstalls) == 0 {
		return Check{Name: "install-sources", Status: "ok", Message: "no local install sources recorded in workspace metadata"}
	}

	var missing []string
	var drifted []string
	var healthy []string
	for _, install := range localInstalls {
		sourcePath := resolveLocalSourcePath(cwd, install.Source)
		if _, err := os.Stat(sourcePath); err != nil {
			missing = append(missing, fmt.Sprintf("%s@%s -> %s", install.Slug, install.Host, filepath.ToSlash(sourcePath)))
			continue
		}

		installedPath := filepath.Join(cwd, filepath.FromSlash(install.InstalledDir))
		if same, err := packageSurfaceEqual(sourcePath, installedPath); err != nil {
			drifted = append(drifted, fmt.Sprintf("%s@%s -> compare error: %v", install.Slug, install.Host, err))
			continue
		} else if !same {
			drifted = append(drifted, fmt.Sprintf("%s@%s", install.Slug, install.Host))
			continue
		}
		healthy = append(healthy, fmt.Sprintf("%s@%s", install.Slug, install.Host))
	}

	if len(missing) > 0 || len(drifted) > 0 {
		var parts []string
		if len(missing) > 0 {
			sort.Strings(missing)
			parts = append(parts, "missing sources: "+strings.Join(missing, ", "))
		}
		if len(drifted) > 0 {
			sort.Strings(drifted)
			parts = append(parts, "installed copy differs from local source: "+strings.Join(drifted, ", "))
		}
		return Check{Name: "install-sources", Status: "warning", Message: strings.Join(parts, "; ")}
	}

	sort.Strings(healthy)
	return Check{
		Name:    "install-sources",
		Status:  "ok",
		Message: fmt.Sprintf("all %d local install source(s) resolve and match installed copies: %s", len(healthy), strings.Join(healthy, ", ")),
	}
}

func filterLocalInstalls(installs []workspace.InstallRecord) []workspace.InstallRecord {
	var local []workspace.InstallRecord
	for _, install := range installs {
		if isLocalSource(install.Source) {
			local = append(local, install)
		}
	}
	return local
}

func isLocalSource(source string) bool {
	source = strings.TrimSpace(source)
	switch {
	case source == "":
		return false
	case filepath.IsAbs(source):
		return true
	case strings.HasPrefix(source, "."):
		return true
	case strings.Contains(source, "://"):
		return false
	case strings.HasPrefix(source, "github.com/"):
		return false
	case strings.HasPrefix(source, "git@"):
		return false
	case strings.Contains(source, "/"), strings.Contains(source, string(filepath.Separator)):
		return true
	default:
		return false
	}
}

func resolveLocalSourcePath(cwd, source string) string {
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}
	return filepath.Join(cwd, filepath.FromSlash(source))
}

func packageSurfaceEqual(sourceRoot, installedRoot string) (bool, error) {
	sourceFingerprint, err := packageSurfaceFingerprint(sourceRoot)
	if err != nil {
		return false, err
	}
	installedFingerprint, err := packageSurfaceFingerprint(installedRoot)
	if err != nil {
		return false, err
	}
	return sourceFingerprint == installedFingerprint, nil
}

func packageSurfaceFingerprint(root string) (string, error) {
	allowed := []string{
		"SKILL.md",
		"EPIC.md",
		"runtime",
		"ROADMAP.md",
		"DECISIONS.md",
		"state.json",
		"state",
		"plans",
		"log",
		"artifacts",
		"hooks",
		"cron.d",
		"policy.yml",
	}

	hasher := sha256.New()
	for _, name := range allowed {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}

		if !info.IsDir() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			_, _ = hasher.Write([]byte(filepath.ToSlash(name)))
			_, _ = hasher.Write(raw)
			continue
		}

		if err := filepath.Walk(path, func(current string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			_, _ = hasher.Write([]byte(filepath.ToSlash(rel)))
			_, _ = hasher.Write(raw)
			return nil
		}); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
