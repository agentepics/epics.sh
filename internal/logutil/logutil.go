package logutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type logFile struct {
	path    string
	modTime time.Time
}

func Recent(root string, limit int) ([]Entry, error) {
	if limit <= 0 {
		return []Entry{}, nil
	}

	logDir := filepath.Join(root, "log")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}

	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		files = append(files, logFile{
			path:    filepath.Join(logDir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].modTime.Equal(files[j].modTime) {
			return filepath.Base(files[i].path) > filepath.Base(files[j].path)
		}
		return files[i].modTime.After(files[j].modTime)
	})

	if len(files) > limit {
		files = files[:limit]
	}

	result := make([]Entry, 0, len(files))
	for _, file := range files {
		raw, err := os.ReadFile(file.path)
		if err != nil {
			return nil, err
		}
		result = append(result, Entry{
			Path:    file.path,
			Content: string(raw),
		})
	}

	return result, nil
}

func Create(root, title string) (string, error) {
	return CreateAt(root, title, time.Now().UTC())
}

func CreateAt(root, title string, now time.Time) (string, error) {
	timestamp := now.UTC()
	cleanTitle := sanitizeTitle(title)

	var name string
	if cleanTitle == "" {
		name = timestamp.Format("2006-01-02-150405") + ".md"
	} else {
		name = timestamp.Format("2006-01-02") + "-" + slugify(cleanTitle) + ".md"
	}

	path := filepath.Join(root, "log", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	content := fmt.Sprintf("---\ndate: %s\ntitle: %s\n---\n", timestamp.Format(time.RFC3339), cleanTitle)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func sanitizeTitle(title string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
}

func slugify(value string) string {
	var b strings.Builder
	lastHyphen := false

	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.Trim(b.String(), "-")
}
