// Package job defines the Job type and ID generation logic.
package job

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Job represents a single ldcron-managed launchd job.
type Job struct {
	// ID is the unique 8-character hex identifier derived from Schedule+Args.
	ID string
	// Label is the launchd service label: com.ldcron.<ID>
	Label string
	// Schedule is the original cron expression.
	Schedule string
	// Args is [command, arg1, arg2, ...].
	Args []string
}

// NewJob creates a Job with a deterministic ID based on schedule and args.
func NewJob(schedule string, args []string) *Job {
	id := generateID(schedule, args)
	return &Job{
		ID:       id,
		Label:    "com.ldcron." + id,
		Schedule: schedule,
		Args:     args,
	}
}

// generateID produces a 16-character hex hash from schedule and args.
func generateID(schedule string, args []string) string {
	key := schedule + "\x00" + strings.Join(args, "\x00")
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum[:8])
}
