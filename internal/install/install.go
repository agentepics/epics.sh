package install

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/workspace"
)

type RegistryEntry struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Source  struct {
		Repo string `json:"repo"`
		Path string `json:"path"`
	} `json:"source"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	SkillMD string `json:"skillMd"`
	EpicMD  string `json:"epicMd"`
}

type InstallResult struct {
	Package     epic.Package            `json:"package"`
	Diagnostics []epic.Diagnostic       `json:"diagnostics,omitempty"`
	Install     workspace.InstallRecord `json:"install"`
	SourceKind  string                  `json:"sourceKind"`
}

type ResolvedSource struct {
	Kind    string
	Input   string
	Root    string
	Entry   *RegistryEntry
	Package epic.Package
	Cleanup func() error
}

func Resolve(cwd, input string) (ResolvedSource, error) {
	if input == "" {
		return ResolvedSource{}, errors.New("install requires a source path")
	}

	localPath := input
	if !filepath.IsAbs(localPath) {
		localPath = filepath.Join(cwd, input)
	}
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		pkg, _, err := epic.Validate(localPath)
		if err != nil {
			return ResolvedSource{}, err
		}
		return ResolvedSource{
			Kind:    "local",
			Input:   input,
			Root:    localPath,
			Package: pkg,
		}, nil
	}

	entry, err := FindRegistryEntry(cwd, input)
	if err == nil {
		return ResolvedSource{
			Kind:  "registry",
			Input: input,
			Entry: &entry,
			Package: epic.Package{
				Slug:    entry.Slug,
				Title:   entry.Title,
				Summary: entry.Summary,
			},
		}, nil
	}

	remote, ok := ParseGitHubSource(input)
	if ok {
		root, cleanup, err := CloneGitHubSource(remote)
		if err != nil {
			return ResolvedSource{}, err
		}

		pkg, _, err := epic.Validate(root)
		if err != nil {
			_ = cleanup()
			return ResolvedSource{}, err
		}

		return ResolvedSource{
			Kind:    "remote",
			Input:   input,
			Root:    root,
			Package: pkg,
			Cleanup: cleanup,
		}, nil
	}

	return ResolvedSource{}, fmt.Errorf("could not resolve source %q as a local directory or registry entry", input)
}

func Install(cwd, input, host string, installDir func(slug string) string) (InstallResult, error) {
	source, err := Resolve(cwd, input)
	if err != nil {
		return InstallResult{}, err
	}
	if source.Cleanup != nil {
		defer source.Cleanup()
	}
	if installDir == nil {
		return InstallResult{}, errors.New("install destination is not configured")
	}

	if err := workspace.EnsureManagedDir(cwd); err != nil {
		return InstallResult{}, err
	}

	dest := installDir(source.Package.Slug)
	if dest == "" {
		return InstallResult{}, errors.New("install destination is empty")
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(cwd, dest)
	}
	installedDir, err := filepath.Rel(cwd, dest)
	if err != nil {
		return InstallResult{}, err
	}
	installedDir = filepath.ToSlash(installedDir)
	if installedDir == "." || strings.HasPrefix(installedDir, "../") {
		return InstallResult{}, errors.New("install destination must stay within the current workspace")
	}

	stagingDir, err := newInstallStagingDir(dest)
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(stagingDir)

	switch source.Kind {
	case "local":
		if err := copyPackageSurface(source.Root, stagingDir); err != nil {
			return InstallResult{}, err
		}
	case "remote":
		if err := copyPackageSurface(source.Root, stagingDir); err != nil {
			return InstallResult{}, err
		}
	case "registry":
		if err := materializeRegistryEntry(stagingDir, *source.Entry); err != nil {
			return InstallResult{}, err
		}
	default:
		return InstallResult{}, fmt.Errorf("unsupported source kind %q", source.Kind)
	}

	pkg, diagnostics, err := epic.Validate(stagingDir)
	if err != nil {
		return InstallResult{}, err
	}
	if source.Package.Slug != "" {
		pkg.Slug = source.Package.Slug
	}
	if source.Package.Title != "" {
		pkg.Title = source.Package.Title
	}
	if source.Package.Summary != "" {
		pkg.Summary = source.Package.Summary
	}
	if source.Package.EpicID != "" {
		pkg.EpicID = source.Package.EpicID
	}
	if epic.HasErrors(diagnostics) {
		return InstallResult{Package: pkg, Diagnostics: diagnostics}, errors.New("installed package did not validate")
	}

	if err := RunInstallHooks(pkg); err != nil {
		return InstallResult{}, err
	}
	if err := promoteInstall(stagingDir, dest); err != nil {
		return InstallResult{}, err
	}

	record := workspace.NewInstallRecord(pkg.Slug, pkg.Title, host, input, sourceVersion(source), sourceDigest(source), installedDir)
	if err := workspace.SaveInstall(cwd, record); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Package:     pkg,
		Diagnostics: diagnostics,
		Install:     record,
		SourceKind:  source.Kind,
	}, nil
}

func FindRegistryEntry(cwd, input string) (RegistryEntry, error) {
	registryRoot, err := findUpward(cwd, filepath.Join("registry", "epics"))
	if err != nil {
		return RegistryEntry{}, err
	}

	entries, err := os.ReadDir(registryRoot)
	if err != nil {
		return RegistryEntry{}, err
	}

	var matches []RegistryEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(registryRoot, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return RegistryEntry{}, err
		}

		var candidate RegistryEntry
		if err := json.Unmarshal(raw, &candidate); err != nil {
			return RegistryEntry{}, err
		}

		sourcePath := strings.Trim(candidate.Source.Repo+"/"+candidate.Source.Path, "/")
		if input == candidate.Slug || input == sourcePath {
			matches = append(matches, candidate)
		}
	}

	if len(matches) == 0 {
		return RegistryEntry{}, fmt.Errorf("registry entry not found for %q", input)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Slug < matches[j].Slug
	})
	return matches[0], nil
}

func sourceVersion(source ResolvedSource) string {
	if source.Entry != nil {
		return source.Entry.Version
	}
	return ""
}

func sourceDigest(source ResolvedSource) string {
	if source.Entry != nil {
		return source.Entry.Digest
	}
	return ""
}

func materializeRegistryEntry(dest string, entry RegistryEntry) error {
	if strings.TrimSpace(entry.SkillMD) == "" || strings.TrimSpace(entry.EpicMD) == "" {
		return errors.New("registry entry is missing inline SKILL.md or EPIC.md content")
	}

	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(strings.TrimSpace(entry.SkillMD)+"\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "EPIC.md"), []byte(strings.TrimSpace(entry.EpicMD)+"\n"), 0o644)
}

func newInstallStagingDir(dest string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	suffix, err := randomSuffix()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".install-"+suffix)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func promoteInstall(stagingDir, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return os.Rename(stagingDir, dest)
}

func randomSuffix() (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func copyPackageSurface(srcRoot, destRoot string) error {
	allowed := []string{
		"SKILL.md",
		"EPIC.md",
		"ROADMAP.md",
		"DECISIONS.md",
		"state.json",
		"state",
		"plans",
		"log",
		"hooks",
		"cron.d",
		"policy.yml",
	}

	for _, name := range allowed {
		srcPath := filepath.Join(srcRoot, name)
		info, err := os.Stat(srcPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}

		destPath := filepath.Join(destRoot, name)
		if info.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(srcPath, destPath, info.Mode()); err != nil {
			return err
		}
	}

	return nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, raw, mode.Perm())
}

func findUpward(start, relative string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(current, relative)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("could not find %s from %s", relative, start)
}
