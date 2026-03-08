package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	managedDirName = ".epics"
	installsFile   = "installs.json"
)

type InstallRecord struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Host         string `json:"host"`
	Source       string `json:"source"`
	Version      string `json:"version,omitempty"`
	Digest       string `json:"digest,omitempty"`
	InstalledAt  string `json:"installedAt"`
	InstalledDir string `json:"installedDir"`
}

type installIndex struct {
	Installs []InstallRecord `json:"installs"`
}

func ManagedDir(cwd string) string {
	return filepath.Join(cwd, managedDirName)
}

func InstallsPath(cwd string) string {
	return filepath.Join(ManagedDir(cwd), installsFile)
}

func EnsureManagedDir(cwd string) error {
	return os.MkdirAll(ManagedDir(cwd), 0o755)
}

func SaveInstall(cwd string, record InstallRecord) error {
	if err := EnsureManagedDir(cwd); err != nil {
		return err
	}

	index, err := loadIndex(cwd)
	if err != nil {
		return err
	}

	replaced := false
	for i := range index.Installs {
		if index.Installs[i].Slug == record.Slug && index.Installs[i].Host == record.Host {
			index.Installs[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		index.Installs = append(index.Installs, record)
	}

	sort.Slice(index.Installs, func(i, j int) bool {
		if index.Installs[i].Slug == index.Installs[j].Slug {
			return index.Installs[i].Host < index.Installs[j].Host
		}
		return index.Installs[i].Slug < index.Installs[j].Slug
	})

	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(InstallsPath(cwd), raw, 0o644)
}

func LoadInstalls(cwd string) ([]InstallRecord, error) {
	index, err := loadIndex(cwd)
	if err != nil {
		return nil, err
	}
	return index.Installs, nil
}

func FindInstall(cwd, slug string) (InstallRecord, bool, error) {
	installs, err := LoadInstalls(cwd)
	if err != nil {
		return InstallRecord{}, false, err
	}
	var matches []InstallRecord
	for _, install := range installs {
		if install.Slug == slug {
			matches = append(matches, install)
		}
	}
	if len(matches) > 1 {
		return InstallRecord{}, false, errors.New("multiple installed hosts match that slug; pass an explicit path instead")
	}
	if len(matches) == 1 {
		return matches[0], true, nil
	}
	return InstallRecord{}, false, nil
}

func NewInstallRecord(slug, title, host, source, version, digest, installedDir string) InstallRecord {
	return InstallRecord{
		Slug:         slug,
		Title:        title,
		Host:         host,
		Source:       source,
		Version:      version,
		Digest:       digest,
		InstalledAt:  time.Now().UTC().Format(time.RFC3339),
		InstalledDir: filepath.ToSlash(filepath.Clean(installedDir)),
	}
}

func loadIndex(cwd string) (installIndex, error) {
	raw, err := os.ReadFile(InstallsPath(cwd))
	if errors.Is(err, os.ErrNotExist) {
		return installIndex{}, nil
	}
	if err != nil {
		return installIndex{}, err
	}

	var index installIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return installIndex{}, err
	}
	if index.Installs == nil {
		index.Installs = []InstallRecord{}
	}
	return index, nil
}
