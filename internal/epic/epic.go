package epic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Diagnostic struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type Package struct {
	Root        string   `json:"root"`
	Slug        string   `json:"slug"`
	SpecVersion string   `json:"specVersion,omitempty"`
	EpicID      string   `json:"epicId,omitempty"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary,omitempty"`
	SkillPath   string   `json:"skillPath,omitempty"`
	EpicPath    string   `json:"epicPath,omitempty"`
	LiveRoot    string   `json:"liveRoot,omitempty"`
	StatePath   string   `json:"statePath,omitempty"`
	StateCore   string   `json:"stateCorePath,omitempty"`
	PlanFiles   []string `json:"planFiles,omitempty"`
	LogFiles    []string `json:"logFiles,omitempty"`
	Roadmap     string   `json:"roadmapPath,omitempty"`
	Decisions   string   `json:"decisionsPath,omitempty"`
	PolicyPath  string   `json:"policyPath,omitempty"`
}

func Load(root string) (Package, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Package{}, err
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return Package{}, err
	}
	if !info.IsDir() {
		return Package{}, fmt.Errorf("%s is not a directory", absRoot)
	}

	pkg := Package{
		Root: absRoot,
		Slug: sanitizeSlug(filepath.Base(absRoot)),
	}

	pkg.SkillPath = firstExisting(absRoot, "SKILL.md")
	pkg.EpicPath = firstExisting(absRoot, "EPIC.md")
	pkg.PolicyPath = firstExisting(absRoot, "policy.yml")

	if frontmatter := parseFrontmatter(readFile(pkg.EpicPath)); len(frontmatter) > 0 {
		pkg.EpicID = strings.TrimSpace(frontmatter["id"])
		pkg.SpecVersion = strings.TrimSpace(frontmatter["spec_version"])
	}
	if pkg.EpicID == "" {
		pkg.EpicID = pkg.Slug
	}
	pkg.LiveRoot = LiveRoot(absRoot, pkg.SpecVersion)
	pkg.StateCore = firstExisting(absRoot, filepath.Join(RelativeLiveRoot(pkg.SpecVersion), "state", "core.json"))
	pkg.StatePath = firstExisting(absRoot, filepath.Join(RelativeLiveRoot(pkg.SpecVersion), "state.json"))
	pkg.Roadmap = firstExisting(absRoot, filepath.Join(RelativeLiveRoot(pkg.SpecVersion), "ROADMAP.md"))
	pkg.Decisions = firstExisting(absRoot, filepath.Join(RelativeLiveRoot(pkg.SpecVersion), "DECISIONS.md"))
	pkg.PlanFiles = collectFiles(filepath.Join(pkg.LiveRoot, "plans"))
	pkg.LogFiles = collectFiles(filepath.Join(pkg.LiveRoot, "log"))

	title := extractHeading(readFile(pkg.EpicPath))
	if isPlaceholderHeading(title) {
		title = ""
	}
	if title == "" {
		title = extractHeading(readFile(pkg.SkillPath))
		if isPlaceholderHeading(title) {
			title = ""
		}
	}
	if title == "" {
		title = humanizeSlug(pkg.Slug)
	}
	pkg.Title = title

	summary := extractSummary(readFile(pkg.EpicPath))
	if summary == "" {
		summary = extractSummary(readFile(pkg.SkillPath))
	}
	pkg.Summary = summary

	return pkg, nil
}

func Validate(root string) (Package, []Diagnostic, error) {
	pkg, err := Load(root)
	if err != nil {
		return Package{}, nil, err
	}

	var diagnostics []Diagnostic

	if pkg.SkillPath == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "missing_skill_md",
			Message: "missing required file SKILL.md",
			Path:    "SKILL.md",
		})
	} else {
		diagnostics = append(diagnostics, validateSkillMarkdown(pkg.Root, pkg.SkillPath, "SKILL.md", pkg.SpecVersion)...)
	}

	if pkg.EpicPath == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "missing_epic_md",
			Message: "missing required file EPIC.md",
			Path:    "EPIC.md",
		})
	} else {
		diagnostics = append(diagnostics, validateMarkdown(pkg.Root, pkg.EpicPath, "EPIC.md", "epic")...)
	}

	diagnostics = append(diagnostics, validateJSONFile(pkg.Root, pkg.StatePath, relativeLivePath(pkg, "state.json"))...)
	diagnostics = append(diagnostics, validateJSONFile(pkg.Root, pkg.StateCore, relativeLivePath(pkg, filepath.Join("state", "core.json")))...)
	diagnostics = append(diagnostics, validateDirectory(pkg.Root, filepath.Join(pkg.LiveRoot, "plans"), relativeLivePath(pkg, "plans"))...)
	diagnostics = append(diagnostics, validateDirectory(pkg.Root, filepath.Join(pkg.LiveRoot, "log"), relativeLivePath(pkg, "log"))...)
	diagnostics = append(diagnostics, validateRuntimeLayout(pkg)...)

	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Level == diagnostics[j].Level {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Level < diagnostics[j].Level
	})

	return pkg, diagnostics, nil
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == "error" {
			return true
		}
	}
	return false
}

func RelativePath(root, path string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func ReadState(pkg Package) (map[string]any, string, error) {
	path := pkg.StateCore
	if path == "" {
		path = pkg.StatePath
	}
	if path == "" {
		return nil, "", nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, "", err
	}
	return data, path, nil
}

func LookupString(data any, keys ...string) string {
	if len(keys) == 0 || data == nil {
		return ""
	}

	keyset := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keyset[strings.ToLower(key)] = struct{}{}
	}

	return lookupString(data, keyset)
}

func lookupString(data any, keys map[string]struct{}) string {
	switch value := data.(type) {
	case map[string]any:
		for key, nested := range value {
			if _, ok := keys[strings.ToLower(key)]; ok {
				if str, ok := nested.(string); ok {
					return strings.TrimSpace(str)
				}
			}
		}
		for _, nested := range value {
			if result := lookupString(nested, keys); result != "" {
				return result
			}
		}
	case []any:
		for _, nested := range value {
			if result := lookupString(nested, keys); result != "" {
				return result
			}
		}
	}

	return ""
}

func ExtractPlanExcerpt(content string) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "## Now") || strings.EqualFold(trimmed, "# Now") {
			var excerpt []string
			for _, next := range lines[i+1:] {
				trimmedNext := strings.TrimSpace(next)
				if strings.HasPrefix(trimmedNext, "#") && len(excerpt) > 0 {
					break
				}
				if trimmedNext == "" {
					continue
				}
				excerpt = append(excerpt, trimmedNext)
				if len(excerpt) == 4 {
					break
				}
			}
			if len(excerpt) > 0 {
				return strings.Join(excerpt, " ")
			}
		}
	}

	var excerpt []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		excerpt = append(excerpt, trimmed)
		if len(excerpt) == 4 {
			break
		}
	}
	return strings.Join(excerpt, " ")
}

func LatestFiles(paths []string, limit int) []string {
	if len(paths) == 0 || limit <= 0 {
		return nil
	}

	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	if len(sorted) <= limit {
		return sorted
	}
	return sorted[len(sorted)-limit:]
}

func UsesRuntimeLayout(specVersion string) bool {
	switch strings.TrimSpace(specVersion) {
	case "0.5.1", "0.5.2":
		return true
	default:
		return false
	}
}

func RelativeLiveRoot(specVersion string) string {
	if UsesRuntimeLayout(specVersion) {
		return "runtime"
	}
	return "."
}

func LiveRoot(root, specVersion string) string {
	if UsesRuntimeLayout(specVersion) {
		return filepath.Join(root, "runtime")
	}
	return root
}

func RuntimePath(root, specVersion string, relative string) string {
	parts := []string{root}
	if UsesRuntimeLayout(specVersion) {
		parts = append(parts, "runtime")
	}
	parts = append(parts, filepath.FromSlash(relative))
	return filepath.Join(parts...)
}

func firstExisting(root string, relative string) string {
	path := filepath.Join(root, relative)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func collectFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	return paths
}

func validateMarkdown(root, path, relPath, prefix string) []Diagnostic {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []Diagnostic{{
			Level:   "error",
			Code:    "read_error_" + prefix,
			Path:    relPath,
			Message: err.Error(),
		}}
	}

	content := strings.TrimSpace(string(raw))
	if content == "" {
		return []Diagnostic{{
			Level:   "error",
			Code:    "empty_" + prefix,
			Path:    relPath,
			Message: relPath + " must not be empty",
		}}
	}

	if extractHeading(content) == "" {
		return []Diagnostic{{
			Level:   "warning",
			Code:    "missing_heading_" + prefix,
			Path:    relPath,
			Message: relPath + " should include a top-level heading",
		}}
	}

	return nil
}

func validateSkillMarkdown(root, path, relPath, specVersion string) []Diagnostic {
	diagnostics := validateMarkdown(root, path, relPath, "skill")
	if !RequiresDualPurposeSkill(specVersion) {
		return diagnostics
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "read_error_skill",
			Path:    relPath,
			Message: err.Error(),
		})
	}

	content := normalizeNewlines(string(raw))
	frontmatter := parseFrontmatter(content)
	if len(frontmatter) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "missing_skill_frontmatter",
			Path:    relPath,
			Message: relPath + " must start with SKILL.md frontmatter for spec_version 0.5.2",
		})
	} else {
		if strings.TrimSpace(frontmatter["name"]) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Level:   "error",
				Code:    "missing_skill_name",
				Path:    relPath,
				Message: relPath + " must declare frontmatter field name for spec_version 0.5.2",
			})
		}
		if strings.TrimSpace(frontmatter["description"]) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Level:   "error",
				Code:    "missing_skill_description",
				Path:    relPath,
				Message: relPath + " must declare frontmatter field description for spec_version 0.5.2",
			})
		}
	}

	if !strings.Contains(content, CanonicalSkillFooterHeading) {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "missing_agent_epics_heading",
			Path:    relPath,
			Message: relPath + " must include the standard `## Agent Epics` footer heading for spec_version 0.5.2",
		})
	}

	switch {
	case strings.Contains(content, CanonicalSkillFooterMarker):
		// Current footer marker is present.
	case strings.Contains(content, "<!-- epics-canonical-footer:"):
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "stale_agent_epics_footer",
			Path:    relPath,
			Message: relPath + " has a stale Agent Epics footer marker; run `epics upgrade-skill-footer` to refresh it",
		})
	default:
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "missing_agent_epics_footer",
			Path:    relPath,
			Message: relPath + " must include the canonical Agent Epics footer marker for spec_version 0.5.2",
		})
	}
	if strings.Contains(content, CanonicalSkillFooterMarker) &&
		strings.Contains(content, CanonicalSkillFooterHeading) &&
		!skillFooterMatchesCanonical(content) {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "error",
			Code:    "stale_agent_epics_footer_body",
			Path:    relPath,
			Message: relPath + " has a non-canonical Agent Epics footer body; run `epics upgrade-skill-footer` to refresh it",
		})
	}

	preface := content
	if headingIndex := footerHeadingIndex(preface); headingIndex >= 0 {
		preface = preface[:headingIndex]
	}
	if !strings.Contains(preface, "EPIC.md") {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "warning",
			Code:    "missing_epic_reference_skill",
			Path:    relPath,
			Message: relPath + " should mention that durable operating context lives in EPIC.md",
		})
	}
	if !strings.Contains(preface, "See the **Agent Epics** section below") {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   "warning",
			Code:    "missing_agent_epics_pointer",
			Path:    relPath,
			Message: relPath + " should include a pointer near the top that directs first-time readers to `## Agent Epics`",
		})
	}

	return diagnostics
}

func validateJSONFile(root, path, relPath string) []Diagnostic {
	if path == "" {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return []Diagnostic{{
			Level:   "error",
			Code:    "read_error_json",
			Path:    relPath,
			Message: err.Error(),
		}}
	}

	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return []Diagnostic{{
			Level:   "error",
			Code:    "invalid_json",
			Path:    relPath,
			Message: err.Error(),
		}}
	}

	if _, ok := data.(map[string]any); !ok {
		return []Diagnostic{{
			Level:   "warning",
			Code:    "json_not_object",
			Path:    relPath,
			Message: relPath + " should contain a JSON object",
		}}
	}

	return nil
}

func validateDirectory(root, path, relPath string) []Diagnostic {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []Diagnostic{{
			Level:   "error",
			Code:    "stat_error",
			Path:    relPath,
			Message: err.Error(),
		}}
	}
	if !info.IsDir() {
		return []Diagnostic{{
			Level:   "error",
			Code:    "not_directory",
			Path:    relPath,
			Message: relPath + " must be a directory when present",
		}}
	}
	return nil
}

func validateRuntimeLayout(pkg Package) []Diagnostic {
	if !UsesRuntimeLayout(pkg.SpecVersion) {
		return nil
	}

	var diagnostics []Diagnostic
	for _, relPath := range []string{
		"state.json",
		filepath.Join("state", "core.json"),
		"plans",
		"log",
		"ROADMAP.md",
		"DECISIONS.md",
		"artifacts",
	} {
		legacyPath := filepath.Join(pkg.Root, filepath.FromSlash(relPath))
		if _, err := os.Stat(legacyPath); err == nil {
			diagnostics = append(diagnostics, Diagnostic{
				Level:   "error",
				Code:    "legacy_live_state_path",
				Path:    filepath.ToSlash(relPath),
				Message: fmt.Sprintf("%s must move under runtime/ for spec_version %s", filepath.ToSlash(relPath), pkg.SpecVersion),
			})
		}
	}
	return diagnostics
}

func relativeLivePath(pkg Package, relPath string) string {
	relPath = filepath.ToSlash(relPath)
	if UsesRuntimeLayout(pkg.SpecVersion) {
		return filepath.ToSlash(filepath.Join("runtime", relPath))
	}
	return relPath
}

func readFile(path string) string {
	if path == "" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func extractHeading(content string) string {
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

func isPlaceholderHeading(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	return trimmed == "epic.md" || trimmed == "skill.md"
}

func extractSummary(content string) string {
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed
	}
	return ""
}

func parseFrontmatter(content string) map[string]string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}

	values := map[string]string{}
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			return values
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		values[strings.TrimSpace(strings.ToLower(key))] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func humanizeSlug(slug string) string {
	parts := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(slug))
	for i := range parts {
		if len(parts[i]) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, " ")
}
