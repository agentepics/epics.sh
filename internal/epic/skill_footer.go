package epic

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	CanonicalSkillFooterVersion = "v0.5.2"
	CanonicalSkillFooterURL     = "https://raw.githubusercontent.com/agentepics/agentepics/refs/heads/main/footer.md"
	CanonicalSkillFooterMarker  = "<!-- epics-canonical-footer: https://raw.githubusercontent.com/agentepics/agentepics/refs/heads/main/footer.md -->"
	CanonicalSkillFooterHeading = "## Agent Epics"
)

//go:embed footer.md
var canonicalSkillFooter string

func RequiresDualPurposeSkill(specVersion string) bool {
	return strings.TrimSpace(specVersion) == "0.5.2"
}

func CanonicalSkillFooter() string {
	return strings.TrimRight(canonicalSkillFooter, "\n")
}

func RefreshSkillFooter(content string) string {
	trimmed := strings.TrimRight(normalizeNewlines(content), "\n")
	base := stripSkillFooter(trimmed)
	if base == "" {
		return CanonicalSkillFooter() + "\n"
	}
	return base + "\n\n" + CanonicalSkillFooter() + "\n"
}

func UpgradeSkillFooter(root string) (string, bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false, err
	}

	skillPath := firstExisting(absRoot, "SKILL.md")
	if skillPath == "" {
		return "", false, fmt.Errorf("missing required file SKILL.md")
	}

	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return "", false, err
	}

	updated := RefreshSkillFooter(string(raw))
	changed := normalizeNewlines(string(raw)) != updated
	if !changed {
		return skillPath, false, nil
	}

	if err := os.WriteFile(skillPath, []byte(updated), 0o644); err != nil {
		return "", false, err
	}
	return skillPath, true, nil
}

func stripSkillFooter(content string) string {
	content = normalizeNewlines(content)
	if content == "" {
		return ""
	}

	cut := len(content)
	if markerIndex := strings.Index(content, CanonicalSkillFooterMarker); markerIndex >= 0 {
		if headingIndex := footerHeadingIndex(content[:markerIndex]); headingIndex >= 0 {
			cut = headingIndex
		} else {
			cut = markerIndex
		}
	} else if headingIndex := footerHeadingIndex(content); headingIndex >= 0 {
		cut = headingIndex
	}

	return strings.TrimRight(content[:cut], "\n")
}

func extractSkillFooter(content string) string {
	content = normalizeNewlines(content)
	if content == "" {
		return ""
	}

	headingIndex := footerHeadingIndex(content)
	if headingIndex < 0 {
		return ""
	}

	return strings.TrimRight(content[headingIndex:], "\n")
}

func skillFooterMatchesCanonical(content string) bool {
	footer := extractSkillFooter(content)
	if footer == "" {
		return false
	}

	return strings.TrimSpace(footer) == strings.TrimSpace(CanonicalSkillFooter())
}

func normalizeNewlines(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func footerHeadingIndex(content string) int {
	content = normalizeNewlines(content)
	offset := 0
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == CanonicalSkillFooterHeading {
			return offset
		}
		offset += len(line)
		if i < len(lines)-1 {
			offset++
		}
	}
	return -1
}
