// Package job defines the Job type and ID generation logic.
package job

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Job represents a launchd job, either created by ldcron or discovered externally.
type Job struct {
	// ID is the unique identifier: 16-char hex for ldcron-managed, full label for external.
	ID string
	// Label is the launchd service label.
	Label string
	// Schedule is the original cron expression (empty for external jobs without X-Ldcron-Schedule).
	Schedule string
	// Args is [command, arg1, arg2, ...].
	Args []string
	// Managed is true for jobs created by ldcron (com.ldcron.* with X-Ldcron-Schedule).
	Managed bool
}

// NewJob creates a Job with a deterministic ID based on schedule and args.
func NewJob(schedule string, args []string) *Job {
	id := generateID(schedule, args)
	return &Job{
		ID:       id,
		Label:    "com.ldcron." + id,
		Schedule: schedule,
		Args:     args,
		Managed:  true,
	}
}

// generateID produces a 16-character hex hash from schedule and args.
func generateID(schedule string, args []string) string {
	key := schedule + "\x00" + strings.Join(args, "\x00")
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum[:8])
}
