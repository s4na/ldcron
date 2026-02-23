// Package cron provides a cron schedule parser that converts cron expressions
// to launchd StartCalendarInterval entries.
package cron

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CalendarEntry represents a single StartCalendarInterval entry for launchd.
// A nil pointer means the field is not specified (matches any value).
type CalendarEntry struct {
	Minute  *int
	Hour    *int
	Day     *int
	Month   *int
	Weekday *int
}

// fieldSpec defines valid range for each cron field.
type fieldSpec struct {
	min int
	max int
}

// Fields order: minute, hour, day, month, weekday.
var fieldSpecs = []fieldSpec{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 31}, // day of month
	{1, 12}, // month
	{0, 7},  // weekday (0 and 7 both mean Sunday)
}

// ParseSchedule parses a 5-field cron expression and returns a list of
// CalendarEntry values suitable for use in a launchd plist.
func ParseSchedule(expr string) ([]CalendarEntry, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron式は5フィールド必要です (分 時 日 月 曜日): %q", expr)
	}

	// expanded[i] == nil means wildcard for that field.
	expanded := make([][]int, 5)
	for i, part := range parts {
		values, wild, err := expandField(part, fieldSpecs[i])
		if err != nil {
			names := []string{"分", "時", "日", "月", "曜日"}
			return nil, fmt.Errorf("%sフィールドが無効: %w", names[i], err)
		}
		if !wild {
			expanded[i] = values
		}
	}

	entries := buildEntries(expanded)
	return entries, nil
}

// expandField parses one cron field and returns its concrete values.
// Returns (nil, true, nil) for a wildcard.
func expandField(s string, spec fieldSpec) ([]int, bool, error) {
	if s == "*" {
		return nil, true, nil
	}

	var allValues []int
	for _, token := range strings.Split(s, ",") {
		vals, wild, err := expandToken(token, spec)
		if err != nil {
			return nil, false, err
		}
		if wild {
			return nil, true, nil
		}
		allValues = append(allValues, vals...)
	}

	allValues = dedup(allValues)
	sort.Ints(allValues)
	return allValues, false, nil
}

// expandToken handles a single token (no commas): plain value, range, or step.
func expandToken(s string, spec fieldSpec) ([]int, bool, error) {
	// Step: */n  or  a-b/n  or  a/n
	if idx := strings.Index(s, "/"); idx >= 0 {
		return expandStep(s, idx, spec)
	}

	// Range: a-b
	if idx := strings.Index(s, "-"); idx >= 0 {
		return expandRange(s, idx, spec)
	}

	// Single value
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, false, fmt.Errorf("無効な値: %q", s)
	}
	// Normalise Sunday: 7 -> 0.
	if spec.max == 7 && v == 7 {
		v = 0
	}
	if v < spec.min || v > spec.max {
		return nil, false, fmt.Errorf("値 %d が範囲外 [%d, %d]", v, spec.min, spec.max)
	}
	return []int{v}, false, nil
}

func expandStep(s string, slashIdx int, spec fieldSpec) ([]int, bool, error) {
	stepStr := s[slashIdx+1:]
	step, err := strconv.Atoi(stepStr)
	if err != nil || step <= 0 {
		return nil, false, fmt.Errorf("無効なステップ: %q", s)
	}

	base := s[:slashIdx]
	lo, hi := spec.min, spec.max
	if base != "*" {
		if dashIdx := strings.Index(base, "-"); dashIdx >= 0 {
			lo, hi, err = parseRange(base, dashIdx, spec)
			if err != nil {
				return nil, false, err
			}
		} else {
			lo, err = strconv.Atoi(base)
			if err != nil || lo < spec.min || lo > spec.max {
				return nil, false, fmt.Errorf("無効な開始値: %q", s)
			}
			hi = spec.max
		}
	}

	var values []int
	for i := lo; i <= hi; i += step {
		values = append(values, i)
	}
	return values, false, nil
}

func expandRange(s string, dashIdx int, spec fieldSpec) ([]int, bool, error) {
	lo, hi, err := parseRange(s, dashIdx, spec)
	if err != nil {
		return nil, false, err
	}
	values := make([]int, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		values = append(values, i)
	}
	return values, false, nil
}

func parseRange(s string, dashIdx int, spec fieldSpec) (int, int, error) {
	loStr := s[:dashIdx]
	hiStr := s[dashIdx+1:]
	lo, err := strconv.Atoi(loStr)
	if err != nil {
		return 0, 0, fmt.Errorf("無効な範囲: %q", s)
	}
	hi, err := strconv.Atoi(hiStr)
	if err != nil {
		return 0, 0, fmt.Errorf("無効な範囲: %q", s)
	}
	if lo < spec.min || hi > spec.max || lo > hi {
		return 0, 0, fmt.Errorf("範囲 %d-%d が境界外 [%d, %d]", lo, hi, spec.min, spec.max)
	}
	return lo, hi, nil
}

// buildEntries constructs the cartesian product of expanded fields.
// Fields with nil slice are treated as wildcards (not included in entries).
func buildEntries(expanded [][]int) []CalendarEntry {
	entries := []CalendarEntry{{}}
	for fieldIdx, values := range expanded {
		if values == nil {
			continue // wildcard
		}
		var next []CalendarEntry
		for _, entry := range entries {
			for _, v := range values {
				e := entry
				v := v
				switch fieldIdx {
				case 0:
					e.Minute = &v
				case 1:
					e.Hour = &v
				case 2:
					e.Day = &v
				case 3:
					e.Month = &v
				case 4:
					if v == 7 {
						v = 0
					}
					e.Weekday = &v
				}
				next = append(next, e)
			}
		}
		entries = next
	}
	return entries
}

func dedup(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := values[:0]
	for _, v := range values {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
