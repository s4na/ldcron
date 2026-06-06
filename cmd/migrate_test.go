package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLogDir_DryRunDoesNotCreateDirectory(t *testing.T) {
	origDryRun := migrateDryRun
	defer func() { migrateDryRun = origDryRun }()
	migrateDryRun = true
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := migrateLogDir()
	if err != nil {
		t.Fatalf("migrateLogDir: %v", err)
	}
	want := filepath.Join(home, "Library", "Logs", "ldcron")
	if got != want {
		t.Fatalf("log dir: got %q, want %q", got, want)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create log directory, stat err=%v", err)
	}
}

func TestMigrateLogDir_NormalRunCreatesDirectory(t *testing.T) {
	origDryRun := migrateDryRun
	defer func() { migrateDryRun = origDryRun }()
	migrateDryRun = false
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := migrateLogDir()
	if err != nil {
		t.Fatalf("migrateLogDir: %v", err)
	}
	want := filepath.Join(home, "Library", "Logs", "ldcron")
	if got != want {
		t.Fatalf("log dir: got %q, want %q", got, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("normal run should create log directory, info=%v err=%v", info, err)
	}
}
