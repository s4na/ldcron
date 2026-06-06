// Package job (continued): store provides discovery of ldcron-managed plist files.
package job

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

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
	return listMatching(pattern)
}

// ListLdcronNamed returns launchd jobs found in com.ldcron.* plist files.
// ParseWarnings contains entries for com.ldcron.* plist files that could not be
// parsed. A returned job is not guaranteed to be Managed; that still depends on
// the plist carrying ldcron schedule metadata.
func ListLdcronNamed(launchAgentsDir string) ([]*Job, []ParseWarning, error) {
	pattern := filepath.Join(launchAgentsDir, "com.ldcron.*.plist")
	return listMatching(pattern)
}

func listMatching(pattern string) ([]*Job, []ParseWarning, error) {
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

// FindDuplicate looks for an already registered ldcron job equivalent to the
// given job. The current deterministic ID wins over the schedule+args fallback.
// The fallback is a migration safety net for legacy plist files that have not
// yet been normalized to the current ID.
func FindDuplicate(launchAgentsDir string, j *Job) (*Job, error) {
	jobs, _, err := List(launchAgentsDir)
	if err != nil {
		return nil, err
	}
	for _, existing := range jobs {
		if existing.ID == j.ID {
			return existing, nil
		}
	}
	for _, existing := range jobs {
		if existing.Managed && j.Managed && existing.Schedule == j.Schedule && slices.Equal(existing.Args, j.Args) {
			return existing, nil
		}
	}
	return nil, nil
}

// PlistPath returns the discovered plist path when available, otherwise the label-derived path.
func PlistPath(launchAgentsDir string, j *Job) string {
	if j.Path != "" {
		return j.Path
	}
	return filepath.Join(launchAgentsDir, j.Label+".plist")
}

// Remove removes a job's plist file.
// For ldcron-managed jobs the file is deleted. For external jobs the file is
// renamed to "<path>.backup_YYYYMMDD_HHMM" so launchd no longer recognises it
// but the original file is preserved. The backup path is returned (empty for
// managed jobs).
func Remove(launchAgentsDir string, j *Job) (string, error) {
	path := PlistPath(launchAgentsDir, j)
	if j.Managed {
		return "", os.Remove(path)
	}
	backupPath := path + time.Now().Format(".backup_20060102_1504")
	return backupPath, os.Rename(path, backupPath)
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
		return nil, fmt.Errorf("ProgramArguments, Program, or BundleProgram not found")
	}

	return &Job{
		ID:       id,
		Label:    label,
		Path:     path,
		Schedule: schedule,
		Args:     args,
		Managed:  managed,
	}, nil
}
