package install

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitHubSource struct {
	RepoURL string
	RepoRef string
	Subpath string
	Branch  string
}

func ParseGitHubSource(input string) (GitHubSource, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return GitHubSource{}, false
	}

	if strings.HasPrefix(trimmed, "github.com/") {
		parts := strings.Split(strings.Trim(trimmed, "/"), "/")
		if len(parts) < 3 || parts[1] == "" || parts[2] == "" {
			return GitHubSource{}, false
		}
		return GitHubSource{
			RepoURL: "https://" + strings.Join(parts[:3], "/") + ".git",
			RepoRef: strings.Join(parts[:3], "/"),
			Subpath: strings.Join(parts[3:], "/"),
			Branch:  "",
		}, true
	}

	if !strings.HasPrefix(trimmed, "https://github.com/") {
		return GitHubSource{}, false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return GitHubSource{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return GitHubSource{}, false
	}

	source := GitHubSource{
		RepoURL: "https://github.com/" + strings.Join(parts[:2], "/") + ".git",
		RepoRef: "github.com/" + strings.Join(parts[:2], "/"),
	}

	switch {
	case len(parts) >= 4 && parts[2] == "tree":
		source.Branch = parts[3]
		if len(parts) > 4 {
			source.Subpath = strings.Join(parts[4:], "/")
		}
	default:
		if len(parts) > 2 {
			source.Subpath = strings.Join(parts[2:], "/")
		}
	}

	return source, true
}

func CloneGitHubSource(source GitHubSource) (string, func() error, error) {
	tempDir, err := os.MkdirTemp("", "epics-install-*")
	if err != nil {
		return "", nil, err
	}

	args := []string{"clone", "--depth", "1"}
	if source.Branch != "" {
		args = append(args, "--branch", source.Branch)
	}
	args = append(args, source.RepoURL, tempDir)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("git clone failed for %s: %s", source.RepoRef, strings.TrimSpace(string(output)))
	}

	root := tempDir
	if source.Subpath != "" {
		root = filepath.Join(tempDir, filepath.FromSlash(source.Subpath))
	}
	info, err := os.Stat(root)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("resolved GitHub path does not exist in %s: %w", source.RepoRef, err)
	}
	if !info.IsDir() {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("resolved GitHub path %s is not a directory", source.Subpath)
	}

	return root, func() error {
		return os.RemoveAll(tempDir)
	}, nil
}
