package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/cron"
	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/s4na/ldcron/internal/plist"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <schedule> <command> [args...]",
	Short: "Add a job with a cron expression and command",
	Long: `Register a launchd job with a cron schedule and command.

The schedule must be a 5-field cron expression (minute hour day month weekday).
The command must be specified as an absolute path.

Note: Step expressions (e.g. */5) trigger at fixed clock times, not relative to
  the registration time.
  Example: "*/5 * * * *" fires at :00, :05, :10 ...
  If registered at 12:03, the first fire is at 12:05 (not 5 minutes later at 12:08).

Examples:
  ldcron add "0 12 * * *" /usr/local/bin/myscript.sh
  ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/script.rb --verbose`,
	Args:         cobra.MinimumNArgs(2),
	SilenceUsage: true,
	RunE:         runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	schedule := args[0]
	programArgs := args[1:]

	// Validate: command must be absolute path.
	command := programArgs[0]
	if !filepath.IsAbs(command) {
		return fmt.Errorf("command must be an absolute path: %q", command)
	}

	// Validate: command must exist.
	if _, err := os.Stat(command); err != nil {
		return fmt.Errorf("command not found: %q", command)
	}

	// Validate cron expression.
	if _, err := cron.ParseSchedule(schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	for _, w := range cron.ValidateSchedule(schedule) {
		fmt.Fprintln(os.Stderr, w)
	}

	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}
	logD, err := logDir()
	if err != nil {
		return err
	}

	j := job.NewJob(schedule, programArgs)

	// Check for duplicate.
	dup, err := job.FindDuplicate(agentsDir, j)
	if err != nil {
		return fmt.Errorf("duplicate check failed: %w", err)
	}
	if dup != nil {
		return fmt.Errorf("job already registered (ID: %s)", dup.ID)
	}

	// Write plist.
	plistPath, err := plist.Write(agentsDir, j.Label, j.Schedule, j.Args, logD)
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	// Register with launchd.
	lc, err := launchctl.New()
	if err != nil {
		if removeErr := os.Remove(plistPath); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete plist file: %v\n  please remove manually: rm %s\n", removeErr, plistPath)
		}
		return fmt.Errorf("failed to initialize launchctl client: %w", err)
	}
	if err := lc.Bootstrap(plistPath); err != nil {
		if removeErr := os.Remove(plistPath); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete plist file: %v\n  please remove manually: rm %s\n", removeErr, plistPath)
		}
		return fmt.Errorf("failed to register with launchctl: %w", err)
	}

	fmt.Printf("Job added\n")
	fmt.Printf("  ID:       %s\n", j.ID)
	fmt.Printf("  Schedule: %s\n", j.Schedule)
	fmt.Printf("  Command:  %s\n", strings.Join(j.Args, " "))
	fmt.Printf("  Log:      %s/%s.log\n", logD, j.ID)
	return nil
}
