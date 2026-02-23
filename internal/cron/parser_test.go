package cron_test

import (
	"testing"

	"github.com/s4na/ldcron/internal/cron"
)

func intPtr(v int) *int { return &v }

func TestParseSchedule_Valid(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantLen int
		check   func(t *testing.T, entries []cron.CalendarEntry)
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
