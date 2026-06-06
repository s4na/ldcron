package migrate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/migrate"
	"github.com/s4na/ldcron/internal/plist"
)

type fakeLaunchctl struct {
	loaded map[string]bool
	calls  []string
}

func newFakeLaunchctl(loaded ...string) *fakeLaunchctl {
	f := &fakeLaunchctl{loaded: make(map[string]bool)}
	for _, label := range loaded {
		f.loaded[label] = true
	}
	return f
}

func (f *fakeLaunchctl) Bootstrap(plistPath string) error {
	label := strings.TrimSuffix(filepath.Base(plistPath), ".plist")
	f.calls = append(f.calls, "bootstrap "+label)
	f.loaded[label] = true
	return nil
}

func (f *fakeLaunchctl) Bootout(label string) error {
	f.calls = append(f.calls, "bootout "+label)
	f.loaded[label] = false
	return nil
}

func (f *fakeLaunchctl) IsLoaded(label string) bool {
	return f.loaded[label]
}

func TestRun_MigratesLoadedLegacyManagedJobToCurrentID(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	schedule := "0 12 * * *"
	args := []string{"/usr/bin/true"}
	legacyLabel := "com.ldcron.deadbeef"
	legacyPath := writePlist(t, dir, legacyLabel, schedule, args, logDir)
	current := job.NewJob(schedule, args)
	if current.ID == "deadbeef" {
		t.Fatalf("test setup unexpectedly produced legacy ID %q", current.ID)
	}

	lc := newFakeLaunchctl(legacyLabel)
	results, warnings, err := migrate.Run(dir, logDir, lc, migrate.Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings: %+v", warnings)
	}
	if len(results) != 1 {
		t.Fatalf("results: got %d, want 1", len(results))
	}
	if results[0].OldID != "deadbeef" || results[0].NewID != current.ID {
		t.Fatalf("result: got %+v", results[0])
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy plist should be removed, stat err=%v", err)
	}

	currentPath := filepath.Join(dir, current.Label+".plist")
	assertPlistInfo(t, currentPath, current.Label, schedule, args)
	if lc.loaded[legacyLabel] {
		t.Fatal("legacy launchd label should be unloaded")
	}
	if !lc.loaded[current.Label] {
		t.Fatal("current launchd label should be loaded")
	}
	assertCalls(t, lc.calls, []string{
		"bootout " + legacyLabel,
		"bootstrap " + current.Label,
	})
}

func TestRun_MigratesUnloadedLegacyPlistWithoutLoadingJob(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	schedule := "0 12 * * *"
	args := []string{"/usr/bin/true"}
	legacyLabel := "com.ldcron.deadbeef"
	writePlist(t, dir, legacyLabel, schedule, args, logDir)
	current := job.NewJob(schedule, args)

	lc := newFakeLaunchctl()
	results, _, err := migrate.Run(dir, logDir, lc, migrate.Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results: got %d, want 1", len(results))
	}
	if len(lc.calls) != 0 {
		t.Fatalf("launchctl calls: got %v, want none", lc.calls)
	}
	assertPlistInfo(t, filepath.Join(dir, current.Label+".plist"), current.Label, schedule, args)
	if lc.loaded[current.Label] {
		t.Fatal("unloaded legacy job should remain unloaded after plist migration")
	}
}

func TestRun_ConsolidatesLegacyJobWhenCurrentPlistAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	schedule := "0 12 * * *"
	args := []string{"/usr/bin/true"}
	legacyLabel := "com.ldcron.deadbeef"
	legacyPath := writePlist(t, dir, legacyLabel, schedule, args, logDir)
	current := job.NewJob(schedule, args)
	currentPath := writePlist(t, dir, current.Label, schedule, args, logDir)

	lc := newFakeLaunchctl(legacyLabel)
	results, _, err := migrate.Run(dir, logDir, lc, migrate.Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results: got %d, want 1", len(results))
	}
	if !results[0].Consolidated {
		t.Fatalf("result should be consolidated: %+v", results[0])
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy plist should be removed, stat err=%v", err)
	}
	assertPlistInfo(t, currentPath, current.Label, schedule, args)
	if !lc.loaded[current.Label] {
		t.Fatal("current launchd label should be loaded after consolidation")
	}
}

func TestRun_ConsolidatesSecondLegacyJobAfterFirstMigration(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	schedule := "0 12 * * *"
	args := []string{"/usr/bin/true"}
	legacyLabel1 := "com.ldcron.aaaaaaaa"
	legacyLabel2 := "com.ldcron.bbbbbbbb"
	legacyPath1 := writePlist(t, dir, legacyLabel1, schedule, args, logDir)
	legacyPath2 := writePlist(t, dir, legacyLabel2, schedule, args, logDir)
	current := job.NewJob(schedule, args)

	lc := newFakeLaunchctl(legacyLabel1, legacyLabel2)
	results, _, err := migrate.Run(dir, logDir, lc, migrate.Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results: got %d, want 2", len(results))
	}
	consolidatedCount := 0
	for _, result := range results {
		if result.Consolidated {
			consolidatedCount++
		}
	}
	if consolidatedCount != 1 {
		t.Fatalf("exactly one legacy plist should consolidate after the current plist is written: %+v", results)
	}
	for _, path := range []string{legacyPath1, legacyPath2} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("legacy plist should be removed: %s stat err=%v", path, err)
		}
	}
	assertPlistInfo(t, filepath.Join(dir, current.Label+".plist"), current.Label, schedule, args)
	if !lc.loaded[current.Label] {
		t.Fatal("current launchd label should be loaded")
	}
}

func TestRun_DryRunDoesNotChangeFilesOrLaunchd(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	schedule := "0 12 * * *"
	args := []string{"/usr/bin/true"}
	legacyLabel := "com.ldcron.deadbeef"
	legacyPath := writePlist(t, dir, legacyLabel, schedule, args, logDir)
	current := job.NewJob(schedule, args)

	lc := newFakeLaunchctl(legacyLabel)
	results, _, err := migrate.Run(dir, logDir, lc, migrate.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results: got %d, want 1", len(results))
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy plist should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, current.Label+".plist")); !os.IsNotExist(err) {
		t.Fatalf("current plist should not be written in dry run, stat err=%v", err)
	}
	if len(lc.calls) != 0 {
		t.Fatalf("launchctl calls: got %v, want none", lc.calls)
	}
}

func writePlist(t *testing.T, dir, label, schedule string, args []string, logDir string) string {
	t.Helper()
	data, err := plist.Generate(label, schedule, args, logDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	path := filepath.Join(dir, label+".plist")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func assertPlistInfo(t *testing.T, path, wantLabel, wantSchedule string, wantArgs []string) {
	t.Helper()
	gotLabel, gotSchedule, gotArgs, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo(%s): %v", path, err)
	}
	if gotLabel != wantLabel {
		t.Fatalf("label: got %q, want %q", gotLabel, wantLabel)
	}
	if gotSchedule != wantSchedule {
		t.Fatalf("schedule: got %q, want %q", gotSchedule, wantSchedule)
	}
	if strings.Join(gotArgs, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("args: got %q, want %q", gotArgs, wantArgs)
	}
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("calls: got %v, want %v", got, want)
	}
}
