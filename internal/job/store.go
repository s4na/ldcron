// Package job (continued): store provides discovery of ldcron-managed plist files.
package job

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/plist"
)

// List returns all ldcron-managed jobs found in launchAgentsDir.
func List(launchAgentsDir string) ([]*Job, error) {
	pattern := filepath.Join(launchAgentsDir, "com.ldcron.*.plist")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var jobs []*Job
	for _, path := range matches {
		j, err := fromPlist(path)
		if err != nil {
			// Skip files we cannot parse (could be manually edited).
			fmt.Fprintf(os.Stderr, "警告: %s のパースをスキップします: %v\n", path, err)
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// Find returns the job with the given ID, or nil if not found.
func Find(launchAgentsDir, id string) (*Job, error) {
	jobs, err := List(launchAgentsDir)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, nil
}

// FindDuplicate looks for a job with the same ID as the given job.
func FindDuplicate(launchAgentsDir string, j *Job) (*Job, error) {
	return Find(launchAgentsDir, j.ID)
}

// PlistPath returns the expected plist file path for a job.
func PlistPath(launchAgentsDir string, j *Job) string {
	return filepath.Join(launchAgentsDir, j.Label+".plist")
}

// Remove deletes the plist file for the given job.
func Remove(launchAgentsDir string, j *Job) error {
	return os.Remove(PlistPath(launchAgentsDir, j))
}

// fromPlist reconstructs a Job from a plist file path.
func fromPlist(path string) (*Job, error) {
	schedule, args, err := plist.ReadSchedule(path)
	if err != nil {
		return nil, err
	}
	// Extract ID from filename: com.ldcron.<id>.plist
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".plist")
	base = strings.TrimPrefix(base, "com.ldcron.")
	if base == "" {
		return nil, os.ErrInvalid
	}
	return &Job{
		ID:       base,
		Label:    "com.ldcron." + base,
		Schedule: schedule,
		Args:     args,
	}, nil
}
