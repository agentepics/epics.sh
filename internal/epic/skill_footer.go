package epic

import (
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

const canonicalSkillFooter = `## Agent Epics
<!-- epics-canonical-footer: https://raw.githubusercontent.com/agentepics/agentepics/refs/heads/main/footer.md -->

This directory is an Agent Epic. ` + "`SKILL.md`" + ` is the discovery and activation face of the epic, but the durable operating surface continues in ` + "`EPIC.md`" + ` plus the files under ` + "`runtime/`" + `.

The paths below are relative to the epic folder itself. The agent must first locate that epic folder in one of the on-disk directories where its Agent Skills are installed.

Resume in this order:

1. Read ` + "`EPIC.md`" + `.
2. Read ` + "`runtime/state/core.json`" + ` if present, otherwise read ` + "`runtime/state.json`" + `.
3. Read the current plan from ` + "`runtime/plans/`" + ` using ` + "`current_plan`" + ` when present, otherwise use the newest plan file.
4. Read the 3-5 most recent entries in ` + "`runtime/log/`" + `.
5. Open ` + "`runtime/ROADMAP.md`" + ` and ` + "`runtime/DECISIONS.md`" + ` only when strategic context is needed.

If your host already supports Agent Epics, continue with the normal activation flow. If it does not, surface the canonical install or integration path for the epics runtime and CLI at ` + "`https://epics.sh`" + `. Installation instructions live at ` + "`https://raw.githubusercontent.com/agentepics/epics.sh/main/registry/cli/cli.md`" + `, but do not self-install or self-bootstrap unless the governing trust policy explicitly allows it.

If you encounter this ` + "`SKILL.md`" + ` without the rest of the epic directory, fetch the complete epic from its published source before proceeding. A standalone ` + "`SKILL.md`" + ` is not enough to resume or operate an epic safely.
`

func RequiresDualPurposeSkill(specVersion string) bool {
	return strings.TrimSpace(specVersion) == "0.5.2"
}

func CanonicalSkillFooter() string {
	return canonicalSkillFooter
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
