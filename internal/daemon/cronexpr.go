package daemon

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func cronMatches(expr string, at time.Time) bool {
	fields, withSeconds := cronFields(expr)
	if len(fields) == 0 {
		return false
	}
	if withSeconds {
		return cronFieldMatches(fields[0], at.Second(), 0, 59, nil) &&
			cronFieldMatches(fields[1], at.Minute(), 0, 59, nil) &&
			cronFieldMatches(fields[2], at.Hour(), 0, 23, nil) &&
			cronFieldMatches(fields[3], at.Day(), 1, 31, nil) &&
			cronFieldMatches(fields[4], int(at.Month()), 1, 12, monthNames()) &&
			cronFieldMatches(fields[5], int(at.Weekday()), 0, 7, dayNames())
	}
	return cronFieldMatches(fields[0], at.Minute(), 0, 59, nil) &&
		cronFieldMatches(fields[1], at.Hour(), 0, 23, nil) &&
		cronFieldMatches(fields[2], at.Day(), 1, 31, nil) &&
		cronFieldMatches(fields[3], int(at.Month()), 1, 12, monthNames()) &&
		cronFieldMatches(fields[4], int(at.Weekday()), 0, 7, dayNames())
}

func cronDueBetween(expr string, from, to time.Time) []time.Time {
	if !from.Before(to) {
		return nil
	}
	fields, withSeconds := cronFields(expr)
	if len(fields) == 0 {
		return nil
	}
	step := time.Minute
	cursor := from.Truncate(time.Minute).Add(time.Minute)
	end := to.Truncate(time.Minute)
	if withSeconds {
		step = time.Second
		cursor = from.Truncate(time.Second).Add(time.Second)
		end = to.Truncate(time.Second)
	}
	var times []time.Time
	for !cursor.After(end) {
		if cronMatches(expr, cursor) {
			times = append(times, cursor)
		}
		cursor = cursor.Add(step)
	}
	return times
}

func cronFields(expr string) ([]string, bool) {
	fields := strings.Fields(strings.TrimSpace(expr))
	switch len(fields) {
	case 5:
		return fields, false
	case 6:
		return fields, true
	default:
		return nil, false
	}
}

func cronFieldMatches(expr string, value, min, max int, names map[string]int) bool {
	for _, part := range strings.Split(expr, ",") {
		if cronPartMatches(strings.TrimSpace(part), value, min, max, names) {
			return true
		}
	}
	return false
}

func cronPartMatches(part string, value, min, max int, names map[string]int) bool {
	if part == "*" {
		return true
	}

	step := 1
	base := part
	if strings.Contains(part, "/") {
		pieces := strings.Split(part, "/")
		if len(pieces) != 2 {
			return false
		}
		n, err := strconv.Atoi(pieces[1])
		if err != nil || n <= 0 {
			return false
		}
		step = n
		base = pieces[0]
	}

	if base == "*" {
		return (value-min)%step == 0
	}
	if strings.Contains(base, "-") {
		pieces := strings.Split(base, "-")
		if len(pieces) != 2 {
			return false
		}
		start, err1 := parseCronValue(pieces[0], names)
		end, err2 := parseCronValue(pieces[1], names)
		if err1 != nil || err2 != nil || start < min || end > max || start > end {
			return false
		}
		if value < start || value > end {
			return false
		}
		return (value-start)%step == 0
	}

	parsed, err := parseCronValue(base, names)
	if err != nil || parsed < min || parsed > max {
		return false
	}
	return parsed == value
}

func parseCronValue(token string, names map[string]int) (int, error) {
	upper := strings.ToUpper(strings.TrimSpace(token))
	if names != nil {
		if value, ok := names[upper]; ok {
			return value, nil
		}
	}
	value, err := strconv.Atoi(upper)
	if err != nil {
		return 0, fmt.Errorf("invalid cron token %q", token)
	}
	return value, nil
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
		"7":   0,
	}
}
