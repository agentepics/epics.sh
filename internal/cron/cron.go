package cron

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/agentepics/epics.sh/internal/epic"
)

func Validate(root string) ([]epic.Diagnostic, error) {
	cronDir := filepath.Join(root, "cron.d")
	entries, err := os.ReadDir(cronDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var diagnostics []epic.Diagnostic
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(cronDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			diagnostics = append(diagnostics, epic.Diagnostic{
				Level:   "error",
				Code:    "read_error_cron",
				Path:    filepath.ToSlash(filepath.Join("cron.d", entry.Name())),
				Message: err.Error(),
			})
			continue
		}

		diagnostics = append(diagnostics, validateFile(root, path, string(raw))...)
	}

	return diagnostics, nil
}

func validateFile(root, path, content string) []epic.Diagnostic {
	relativePath := filepath.ToSlash(filepath.Join("cron.d", filepath.Base(path)))
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var diagnostics []epic.Diagnostic

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) < 6 {
			diagnostics = append(diagnostics, epic.Diagnostic{
				Level:   "error",
				Code:    "invalid_cron_expression",
				Path:    relativePath,
				Message: fmt.Sprintf("line %d: expected 5 cron fields plus a command", i+1),
			})
			continue
		}

		schedule := fields[:5]
		if err := validateSchedule(schedule); err != nil {
			diagnostics = append(diagnostics, epic.Diagnostic{
				Level:   "error",
				Code:    "invalid_cron_expression",
				Path:    relativePath,
				Message: fmt.Sprintf("line %d: %v", i+1, err),
			})
			continue
		}

		command := fields[5]
		if !commandExists(root, command) {
			diagnostics = append(diagnostics, epic.Diagnostic{
				Level:   "warning",
				Code:    "missing_cron_command",
				Path:    relativePath,
				Message: fmt.Sprintf("line %d: command %q was not found", i+1, command),
			})
		}
	}

	return diagnostics
}

func validateSchedule(fields []string) error {
	if len(fields) != 5 {
		return fmt.Errorf("expected 5 cron fields")
	}

	if err := validateField(fields[0], 0, 59, nil); err != nil {
		return fmt.Errorf("invalid minute field %q: %w", fields[0], err)
	}
	if err := validateField(fields[1], 0, 23, nil); err != nil {
		return fmt.Errorf("invalid hour field %q: %w", fields[1], err)
	}
	if err := validateField(fields[2], 1, 31, nil); err != nil {
		return fmt.Errorf("invalid day-of-month field %q: %w", fields[2], err)
	}
	if err := validateField(fields[3], 1, 12, monthNames()); err != nil {
		return fmt.Errorf("invalid month field %q: %w", fields[3], err)
	}
	if err := validateField(fields[4], 0, 7, dayNames()); err != nil {
		return fmt.Errorf("invalid day-of-week field %q: %w", fields[4], err)
	}

	return nil
}

func validateField(expr string, min, max int, names map[string]int) error {
	parts := strings.Split(expr, ",")
	for _, part := range parts {
		if err := validatePart(strings.TrimSpace(part), min, max, names); err != nil {
			return err
		}
	}
	return nil
}

func validatePart(part string, min, max int, names map[string]int) error {
	if part == "" {
		return fmt.Errorf("empty field part")
	}

	base := part
	if strings.Contains(part, "/") {
		pieces := strings.Split(part, "/")
		if len(pieces) != 2 || pieces[0] == "" || pieces[1] == "" {
			return fmt.Errorf("invalid step syntax")
		}
		step, err := strconv.Atoi(pieces[1])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step value")
		}
		base = pieces[0]
	}

	if base == "*" {
		return nil
	}

	if strings.Contains(base, "-") {
		bounds := strings.Split(base, "-")
		if len(bounds) != 2 || bounds[0] == "" || bounds[1] == "" {
			return fmt.Errorf("invalid range syntax")
		}
		start, err := parseValue(bounds[0], names)
		if err != nil {
			return err
		}
		end, err := parseValue(bounds[1], names)
		if err != nil {
			return err
		}
		if start < min || start > max || end < min || end > max {
			return fmt.Errorf("range out of bounds")
		}
		if start > end {
			return fmt.Errorf("range start exceeds end")
		}
		return nil
	}

	value, err := parseValue(base, names)
	if err != nil {
		return err
	}
	if value < min || value > max {
		return fmt.Errorf("value out of bounds")
	}
	return nil
}

func parseValue(token string, names map[string]int) (int, error) {
	upper := strings.ToUpper(token)
	if names != nil {
		if value, ok := names[upper]; ok {
			return value, nil
		}
	}

	value, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", token)
	}
	return value, nil
}

func commandExists(root, command string) bool {
	if filepath.IsAbs(command) {
		_, err := os.Stat(command)
		return err == nil
	}

	localPath := filepath.Join(root, filepath.FromSlash(command))
	if _, err := os.Stat(localPath); err == nil {
		return true
	}

	_, err := exec.LookPath(command)
	return err == nil
}

func monthNames() map[string]int {
	return map[string]int{
		"JAN": 1,
		"FEB": 2,
		"MAR": 3,
		"APR": 4,
		"MAY": 5,
		"JUN": 6,
		"JUL": 7,
		"AUG": 8,
		"SEP": 9,
		"OCT": 10,
		"NOV": 11,
		"DEC": 12,
	}
}

func dayNames() map[string]int {
	return map[string]int{
		"SUN": 0,
		"MON": 1,
		"TUE": 2,
		"WED": 3,
		"THU": 4,
		"FRI": 5,
		"SAT": 6,
	}
}
