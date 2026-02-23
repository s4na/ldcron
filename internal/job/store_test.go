package job_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/plist"
)

func setupTestDir(t *testing.T, jobs ...*job.Job) string {
	t.Helper()
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	for _, j := range jobs {
		data, err := plist.Generate(j.Label, j.Schedule, j.Args, logDir)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		path := filepath.Join(dir, j.Label+".plist")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return dir
}

func TestList_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	jobs, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestList_ReturnsAllJobs(t *testing.T) {
	j1 := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	j2 := job.NewJob("0 13 * * *", []string{"/usr/bin/bar"})
	dir := setupTestDir(t, j1, j2)

	jobs, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestList_SkipsNonLdcronPlists(t *testing.T) {
	dir := t.TempDir()
	// Write a non-ldcron plist; it should not be picked up by List.
	if err := os.WriteFile(filepath.Join(dir, "com.apple.foo.plist"), []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobs, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestFind_ExistingJob(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dir := setupTestDir(t, j)

	found, err := job.Find(dir, j.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found == nil {
		t.Fatal("expected job, got nil")
	}
	if found.ID != j.ID {
		t.Errorf("ID: got %q, want %q", found.ID, j.ID)
	}
}

func TestFind_MissingJob(t *testing.T) {
	dir := t.TempDir()
	found, err := job.Find(dir, "nonexistent")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}

func TestFindDuplicate_DetectsDuplicate(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dir := setupTestDir(t, j)

	// Same schedule+args → same ID
	j2 := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dup, err := job.FindDuplicate(dir, j2)
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup == nil {
		t.Error("expected duplicate, got nil")
	}
}

func TestFindDuplicate_NoDuplicateForDifferentJob(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dir := setupTestDir(t, j)

	j2 := job.NewJob("0 13 * * *", []string{"/usr/bin/foo"})
	dup, err := job.FindDuplicate(dir, j2)
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup != nil {
		t.Error("expected nil duplicate")
	}
}

func TestRemove_DeletesPlistFile(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dir := setupTestDir(t, j)

	if err := job.Remove(dir, j); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(job.PlistPath(dir, j)); !os.IsNotExist(err) {
		t.Error("plist file should have been deleted")
	}
}
