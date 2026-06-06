// Package migrate normalizes legacy ldcron plist files to the current format.
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/plist"
)

// Launchctl is the subset of launchctl.Client used during migration.
type Launchctl interface {
	Bootstrap(plistPath string) error
	Bootout(label string) error
	IsLoaded(label string) bool
}

// Options controls migration behavior.
type Options struct {
	DryRun bool
}

// Result describes one migrated or consolidated plist.
type Result struct {
	OldID        string
	NewID        string
	OldPath      string
	NewPath      string
	Consolidated bool
	WasLoaded    bool
}

// Run rewrites ldcron-managed plist files whose ID no longer matches the
// current deterministic ID. Loaded jobs are moved from the legacy launchd label
// to the current label; unloaded plist files are only rewritten on disk.
func Run(launchAgentsDir, logDir string, lc Launchctl, opts Options) ([]Result, []job.ParseWarning, error) {
	jobs, warnings, err := job.ListManaged(launchAgentsDir)
	if err != nil {
		return nil, nil, err
	}

	var results []Result
	for _, existing := range jobs {
		if !needsMigration(existing) {
			continue
		}
		result, err := migrateOne(launchAgentsDir, logDir, lc, opts, existing, jobs)
		if err != nil {
			return results, warnings, err
		}
		results = append(results, result)
		if !opts.DryRun && !result.Consolidated {
			current := job.NewJob(existing.Schedule, existing.Args)
			current.Path = result.NewPath
			jobs = append(jobs, current)
		}
	}
	return results, warnings, nil
}

func needsMigration(j *job.Job) bool {
	if !j.Managed || j.Schedule == "" {
		return false
	}
	current := job.NewJob(j.Schedule, j.Args)
	return j.ID != current.ID || j.Label != current.Label || filepath.Base(j.Path) != current.Label+".plist"
}

func migrateOne(launchAgentsDir, logDir string, lc Launchctl, opts Options, legacy *job.Job, allJobs []*job.Job) (Result, error) {
	current := job.NewJob(legacy.Schedule, legacy.Args)
	newPath := filepath.Join(launchAgentsDir, current.Label+".plist")
	result := Result{
		OldID:   legacy.ID,
		NewID:   current.ID,
		OldPath: legacy.Path,
		NewPath: newPath,
	}

	if target := findByPath(allJobs, newPath); target != nil {
		if !sameManagedJob(target, current) {
			return result, fmt.Errorf("cannot migrate %s: target plist already exists with different job: %s", legacy.ID, newPath)
		}
		result.Consolidated = true
		result.WasLoaded = isLoaded(lc, legacy.Label)
		if opts.DryRun {
			return result, nil
		}
		if err := moveLaunchdLabel(lc, legacy, target.Path, result.WasLoaded); err != nil {
			return result, err
		}
		if err := os.Remove(legacy.Path); err != nil {
			return result, fmt.Errorf("failed to remove legacy plist %s: %w", legacy.Path, err)
		}
		return result, nil
	}

	if _, err := os.Stat(newPath); err == nil {
		return result, fmt.Errorf("cannot migrate %s: target plist already exists but could not be parsed: %s", legacy.ID, newPath)
	} else if !os.IsNotExist(err) {
		return result, fmt.Errorf("cannot inspect target plist %s: %w", newPath, err)
	}

	result.WasLoaded = isLoaded(lc, legacy.Label)
	if opts.DryRun {
		return result, nil
	}

	data, err := plist.Generate(current.Label, current.Schedule, current.Args, logDir)
	if err != nil {
		return result, fmt.Errorf("failed to generate migrated plist for %s: %w", legacy.ID, err)
	}

	tmpPath, err := writeTempPlist(newPath, data)
	if err != nil {
		return result, err
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if result.WasLoaded {
		if err := lc.Bootout(legacy.Label); err != nil {
			return result, fmt.Errorf("failed to unload legacy job %s: %w", legacy.ID, err)
		}
	}
	if err := os.Rename(tmpPath, newPath); err != nil {
		if result.WasLoaded {
			_ = lc.Bootstrap(legacy.Path)
		}
		return result, fmt.Errorf("failed to move migrated plist into place: %w", err)
	}
	cleanupTemp = false

	if result.WasLoaded {
		if err := lc.Bootstrap(newPath); err != nil {
			_ = os.Remove(newPath)
			_ = lc.Bootstrap(legacy.Path)
			return result, fmt.Errorf("failed to load migrated job %s: %w", current.ID, err)
		}
	}
	if err := os.Remove(legacy.Path); err != nil {
		return result, fmt.Errorf("failed to remove legacy plist %s: %w", legacy.Path, err)
	}
	return result, nil
}

func findByPath(jobs []*job.Job, path string) *job.Job {
	for _, j := range jobs {
		if j.Path == path {
			return j
		}
	}
	return nil
}

func sameManagedJob(existing, current *job.Job) bool {
	return existing.Managed &&
		existing.ID == current.ID &&
		existing.Label == current.Label &&
		existing.Schedule == current.Schedule &&
		slices.Equal(existing.Args, current.Args)
}

func isLoaded(lc Launchctl, label string) bool {
	return lc != nil && lc.IsLoaded(label)
}

func moveLaunchdLabel(lc Launchctl, legacy *job.Job, currentPath string, legacyLoaded bool) error {
	if !legacyLoaded {
		return nil
	}
	if err := lc.Bootout(legacy.Label); err != nil {
		return fmt.Errorf("failed to unload legacy job %s: %w", legacy.ID, err)
	}
	currentLabel := strings.TrimSuffix(filepath.Base(currentPath), ".plist")
	if !isLoaded(lc, currentLabel) {
		if err := lc.Bootstrap(currentPath); err != nil {
			_ = lc.Bootstrap(legacy.Path)
			return fmt.Errorf("failed to load current job for migrated legacy job %s: %w", legacy.ID, err)
		}
	}
	return nil
}

func writeTempPlist(targetPath string, data []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write temporary migrated plist: %w", err)
	}
	return tmpPath, nil
}
