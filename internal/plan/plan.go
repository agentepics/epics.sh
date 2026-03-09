package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/state"
)

type Entry struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

func List(cwd string) ([]Entry, error) {
	specVersion := detectSpecVersion(cwd)
	dir := epic.RuntimePath(cwd, specVersion, "plans")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var plans []Entry
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		relPath := epic.RelativePath(cwd, filepath.Join(dir, entry.Name()))
		title, err := readTitle(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		plans = append(plans, Entry{Path: relPath, Title: title})
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Path < plans[j].Path
	})
	return plans, nil
}

func Current(cwd string) (Entry, string, error) {
	if snapshot, err := state.Read(cwd); err == nil {
		if ref := epic.LookupString(
			snapshot.Data,
			"currentPlan",
			"current_plan",
			"currentPlanPath",
			"current_plan_path",
			"plan",
			"planPath",
		); ref != "" {
			path := ref
			if !filepath.IsAbs(path) {
				path = filepath.Join(cwd, filepath.FromSlash(ref))
			}
			if raw, err := os.ReadFile(path); err == nil {
				title, titleErr := readTitle(path)
				if titleErr != nil {
					return Entry{}, "", titleErr
				}
				return Entry{
					Path:  epic.RelativePath(cwd, path),
					Title: title,
				}, string(raw), nil
			}
		}
	}

	plans, err := List(cwd)
	if err != nil {
		return Entry{}, "", err
	}
	if len(plans) == 0 {
		return Entry{}, "", errors.New("no plan files found")
	}

	current := plans[len(plans)-1]
	raw, err := os.ReadFile(filepath.Join(cwd, filepath.FromSlash(current.Path)))
	if err != nil {
		return Entry{}, "", err
	}
	return current, string(raw), nil
}

func Create(cwd, title string) (Entry, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled Plan"
	}

	number, err := nextNumber(cwd)
	if err != nil {
		return Entry{}, err
	}
	slug := slugify(title)
	if slug == "" {
		slug = "plan"
	}

	specVersion := detectSpecVersion(cwd)
	path := epic.RuntimePath(cwd, specVersion, filepath.ToSlash(filepath.Join("plans", fmt.Sprintf("%03d-%s.md", number, slug))))
	relPath := epic.RelativePath(cwd, path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Entry{}, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()

	if _, err := file.WriteString("# " + title + "\n"); err != nil {
		return Entry{}, err
	}

	return Entry{Path: relPath, Title: title}, nil
}

func detectSpecVersion(cwd string) string {
	pkg, err := epic.Load(cwd)
	if err != nil {
		return ""
	}
	return pkg.SpecVersion
}

func nextNumber(cwd string) (int, error) {
	plans, err := List(cwd)
	if err != nil {
		return 0, err
	}

	maxNumber := 0
	for _, entry := range plans {
		base := filepath.Base(entry.Path)
		parts := strings.SplitN(base, "-", 2)
		if len(parts) < 2 {
			continue
		}
		number, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if number > maxNumber {
			maxNumber = number
		}
	}
	return maxNumber + 1, nil
}

func readTitle(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "#") {
		return strings.TrimSpace(strings.TrimLeft(first, "#")), nil
	}
	return "", nil
}

func slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))

	var b strings.Builder
	lastDash := false
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || !unicode.IsLetter(r) && !unicode.IsDigit(r):
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}
