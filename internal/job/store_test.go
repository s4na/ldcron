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
	jobs, warnings, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestList_ReturnsAllJobs(t *testing.T) {
	j1 := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	j2 := job.NewJob("0 13 * * *", []string{"/usr/bin/bar"})
	dir := setupTestDir(t, j1, j2)

	jobs, warnings, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestList_IncludesExternalPlists(t *testing.T) {
	dir := t.TempDir()
	// Write a minimal external plist (no X-Ldcron-Schedule).
	externalPlist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
	<key>Label</key><string>com.apple.foo</string>
	<key>ProgramArguments</key><array><string>/usr/bin/foo</string></array>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(dir, "com.apple.foo.plist"), []byte(externalPlist), 0o644); err != nil {
		t.Fatal(err)
	}

	jobs, warnings, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
	if jobs[0].ID != "com.apple.foo" {
		t.Errorf("ID: got %q, want %q", jobs[0].ID, "com.apple.foo")
	}
	if jobs[0].Managed {
		t.Error("expected Managed=false for external job")
	}
	if jobs[0].Schedule != "" {
		t.Errorf("Schedule: got %q, want empty", jobs[0].Schedule)
	}
}

func TestList_ManagedFlagSetForLdcronJobs(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	dir := setupTestDir(t, j)

	jobs, _, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if !jobs[0].Managed {
		t.Error("expected Managed=true for ldcron job")
	}
}

func TestList_MalformedPlistReturnsWarning(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.plist")
	if err := os.WriteFile(badPath, []byte("not xml"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobs, warnings, err := job.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs for malformed plist, got %d", len(jobs))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Path != badPath {
		t.Errorf("warning path: got %q, want %q", warnings[0].Path, badPath)
	}
	if warnings[0].Err == nil {
		t.Error("expected non-nil error in warning")
	}
}

func TestFind_ExternalJobByLabel(t *testing.T) {
	dir := t.TempDir()
	externalPlist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>Label</key><string>com.example.myjob</string>
	<key>ProgramArguments</key><array><string>/usr/bin/myjob</string></array>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(dir, "com.example.myjob.plist"), []byte(externalPlist), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := job.Find(dir, "com.example.myjob")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found == nil {
		t.Fatal("expected job, got nil")
	}
	if found.Label != "com.example.myjob" {
		t.Errorf("Label: got %q, want %q", found.Label, "com.example.myjob")
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
