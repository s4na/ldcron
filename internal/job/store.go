// Package job (continued): store provides discovery of ldcron-managed plist files.
package job

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/plist"
)

// ParseWarning represents a plist file that could not be parsed.
type ParseWarning struct {
	Err  error
	Path string
}

// List returns all launchd jobs found in launchAgentsDir, including both
// ldcron-managed jobs and any other existing plist files.
// ParseWarnings contains entries for plist files that could not be parsed.
func List(launchAgentsDir string) ([]*Job, []ParseWarning, error) {
	pattern := filepath.Join(launchAgentsDir, "*.plist")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, err
	}

	var jobs []*Job
	var warnings []ParseWarning
	for _, path := range matches {
		j, err := fromPlist(path)
		if err != nil {
			warnings = append(warnings, ParseWarning{Path: path, Err: err})
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, warnings, nil
}

// Find returns the job with the given ID, or nil if not found.
// Parse warnings for broken plist files are silently ignored.
func Find(launchAgentsDir, id string) (*Job, error) {
	jobs, _, err := List(launchAgentsDir)
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
// For ldcron-managed plists (com.ldcron.* with X-Ldcron-Schedule), the short
// hex ID is extracted from the filename. For all other plists, the full launchd
// label is used as the ID.
func fromPlist(path string) (*Job, error) {
	label, schedule, args, err := plist.ReadPlistInfo(path)
	if err != nil {
		return nil, err
	}

	// Determine whether this is an ldcron-managed job.
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".plist")
	managed := strings.HasPrefix(base, "com.ldcron.") && schedule != ""

	var id string
	if managed {
		id = strings.TrimPrefix(base, "com.ldcron.")
		if id == "" {
			// safety net: unreachable unless filename is "com.ldcron..plist"
			id = label
		}
	} else {
		id = label
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("ProgramArgumentsが見つかりません")
	}

	return &Job{
		ID:       id,
		Label:    label,
		Schedule: schedule,
		Args:     args,
		Managed:  managed,
	}, nil
}
