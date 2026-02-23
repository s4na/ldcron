package cron_test

import (
	"testing"

	"github.com/s4na/ldcron/internal/cron"
)

func TestParseSchedule_Valid(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		check   func(t *testing.T, entries []cron.CalendarEntry)
		wantLen int
	}{
		{
			name:    "fixed hour and minute",
			expr:    "0 12 * * *",
			wantLen: 1,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				e := entries[0]
				if e.Minute == nil || *e.Minute != 0 {
					t.Errorf("Minute: got %v, want 0", e.Minute)
				}
				if e.Hour == nil || *e.Hour != 12 {
					t.Errorf("Hour: got %v, want 12", e.Hour)
				}
				if e.Day != nil || e.Month != nil || e.Weekday != nil {
					t.Errorf("wildcard fields should be nil")
				}
			},
		},
		{
			name:    "all wildcards",
			expr:    "* * * * *",
			wantLen: 1,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				e := entries[0]
				if e.Minute != nil || e.Hour != nil || e.Day != nil || e.Month != nil || e.Weekday != nil {
					t.Errorf("all fields should be nil for all-wildcard")
				}
			},
		},
		{
			name:    "step */5 in minute",
			expr:    "*/5 * * * *",
			wantLen: 12,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				for i, e := range entries {
					if e.Minute == nil || *e.Minute != i*5 {
						t.Errorf("entry[%d].Minute: got %v, want %d", i, e.Minute, i*5)
					}
				}
			},
		},
		{
			name:    "range 9-17 in hour",
			expr:    "0 9-17 * * *",
			wantLen: 9,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				for i, e := range entries {
					if e.Hour == nil || *e.Hour != 9+i {
						t.Errorf("entry[%d].Hour: got %v, want %d", i, e.Hour, 9+i)
					}
					if e.Minute == nil || *e.Minute != 0 {
						t.Errorf("entry[%d].Minute: got %v, want 0", i, e.Minute)
					}
				}
			},
		},
		{
			name:    "comma list in minute",
			expr:    "0,30 * * * *",
			wantLen: 2,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				if *entries[0].Minute != 0 || *entries[1].Minute != 30 {
					t.Errorf("unexpected minutes: %v, %v", entries[0].Minute, entries[1].Minute)
				}
			},
		},
		{
			name:    "weekday 7 normalized to 0 (Sunday)",
			expr:    "0 0 * * 7",
			wantLen: 1,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				if entries[0].Weekday == nil || *entries[0].Weekday != 0 {
					t.Errorf("Weekday: got %v, want 0", entries[0].Weekday)
				}
			},
		},
		{
			name:    "weekday range 1-5 (Mon-Fri)",
			expr:    "0 9 * * 1-5",
			wantLen: 5,
		},
		{
			name:    "cartesian product: hour and weekday",
			expr:    "0 9-10 * * 1-2",
			wantLen: 4, // 2 hours × 2 weekdays
		},
		{
			name:    "step with range: 0-30/10",
			expr:    "0-30/10 * * * *",
			wantLen: 4, // 0, 10, 20, 30
		},
		{
			// cron OR semantics: both day-of-month and weekday specified →
			// fire on the 15th OR on Monday, not the intersection.
			name:    "day-of-month and weekday OR semantics",
			expr:    "0 0 15 * 1",
			wantLen: 2, // {Minute:0,Hour:0,Day:15} + {Minute:0,Hour:0,Weekday:1}
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				hasDayEntry, hasWeekdayEntry := false, false
				for _, e := range entries {
					if e.Day != nil && *e.Day == 15 && e.Weekday == nil {
						hasDayEntry = true
					}
					if e.Weekday != nil && *e.Weekday == 1 && e.Day == nil {
						hasWeekdayEntry = true
					}
				}
				if !hasDayEntry {
					t.Error("day-of-month entry (Day:15, no Weekday) not found")
				}
				if !hasWeekdayEntry {
					t.Error("weekday entry (Weekday:1, no Day) not found")
				}
			},
		},
		{
			// Multiple days + multiple weekdays: OR semantics.
			name:    "multiple days and weekdays OR semantics",
			expr:    "0 9 1,15 * 1-5",
			wantLen: 7, // 2 day-of-month entries + 5 weekday entries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := cron.ParseSchedule(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != tt.wantLen {
				t.Errorf("len(entries): got %d, want %d", len(entries), tt.wantLen)
			}
			if tt.check != nil {
				tt.check(t, entries)
			}
		})
	}
}

func TestParseSchedule_Macros(t *testing.T) {
	tests := []struct {
		macro   string
		check   func(t *testing.T, entries []cron.CalendarEntry)
		wantLen int
	}{
		{
			macro:   "@hourly",
			wantLen: 1,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				e := entries[0]
				if e.Minute == nil || *e.Minute != 0 {
					t.Errorf("Minute: got %v, want 0", e.Minute)
				}
				if e.Hour != nil {
					t.Errorf("Hour should be wildcard (nil), got %v", e.Hour)
				}
			},
		},
		{
			macro:   "@daily",
			wantLen: 1,
			check: func(t *testing.T, entries []cron.CalendarEntry) {
				e := entries[0]
				if e.Minute == nil || *e.Minute != 0 {
					t.Errorf("Minute: got %v, want 0", e.Minute)
				}
				if e.Hour == nil || *e.Hour != 0 {
					t.Errorf("Hour: got %v, want 0", e.Hour)
				}
			},
		},
		{macro: "@midnight", wantLen: 1},
		{macro: "@weekly", wantLen: 1},
		{macro: "@monthly", wantLen: 1},
		{macro: "@yearly", wantLen: 1},
		{macro: "@annually", wantLen: 1},
	}

	for _, tt := range tests {
		t.Run(tt.macro, func(t *testing.T) {
			entries, err := cron.ParseSchedule(tt.macro)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != tt.wantLen {
				t.Errorf("len(entries): got %d, want %d", len(entries), tt.wantLen)
			}
			if tt.check != nil {
				tt.check(t, entries)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		expr     string
		hasWarn  bool
	}{
		{"0 0 31 2 *", true},  // Feb 31 does not exist
		{"0 0 30 2 *", true},  // Feb 30 does not exist
		{"0 0 29 2 *", false}, // Feb 29 exists (leap year)
		{"0 0 31 1 *", false}, // Jan 31 exists
		{"0 0 31 * *", false}, // wildcard month — no warning
		{"0 0 * 2 *", false},  // wildcard day — no warning
		{"0 0 1 1 *", false},  // normal date
		{"@daily", false},     // macro — no warning
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			warnings := cron.ValidateSchedule(tt.expr)
			if tt.hasWarn && len(warnings) == 0 {
				t.Errorf("expected warning for %q, got none", tt.expr)
			}
			if !tt.hasWarn && len(warnings) > 0 {
				t.Errorf("unexpected warnings for %q: %v", tt.expr, warnings)
			}
		})
	}
}

func TestParseSchedule_Invalid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"wrong field count: 4 fields", "0 12 * *"},
		{"wrong field count: 6 fields", "0 12 * * * *"},
		{"minute out of range", "99 * * * *"},
		{"hour out of range", "0 25 * * *"},
		{"negative minute", "-1 * * * *"},
		{"month out of range", "0 0 * 13 *"},
		{"weekday out of range", "0 0 * * 8"},
		{"invalid step", "*/0 * * * *"},
		{"non-numeric field", "abc * * * *"},
		{"invalid range: lo > hi", "30-10 * * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cron.ParseSchedule(tt.expr)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.expr)
			}
		})
	}
}
