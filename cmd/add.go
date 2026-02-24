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
	Use:   "add <schedule> <command|script> [args...]",
	Short: "Add a job with a cron expression and command",
	Long: `Register a launchd job with a cron schedule and command.

The schedule must be a 5-field cron expression (minute hour day month weekday).

The command must be specified as an absolute path, OR as a single inline shell
script (which is automatically wrapped in /bin/sh -c). Multi-line scripts are
supported when passed as a single quoted argument.

Note: Step expressions (e.g. */5) trigger at fixed clock times, not relative to
  the registration time.
  Example: "*/5 * * * *" fires at :00, :05, :10 ...
  If registered at 12:03, the first fire is at 12:05 (not 5 minutes later at 12:08).

Examples:
  ldcron add "0 12 * * *" /usr/local/bin/myscript.sh
  ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/script.rb --verbose
  ldcron add "0 * * * *" "echo hello && date >> /tmp/log.txt"
  ldcron add "0 * * * *" $'cd /tmp\necho hello\ndate >> log.txt'`,
	Args:         cobra.MinimumNArgs(2),
	SilenceUsage: true,
	RunE:         runAdd,
}

// validateInlineScript checks for content that cannot survive the plist
// XML round-trip or that launchd/shell would mishandle.
func validateInlineScript(script string) error {
	if script == "" {
		return fmt.Errorf("inline script must not be empty")
	}
	for i, ch := range script {
		// XML 1.0 forbids null characters (U+0000); they cannot be represented
		// in a plist and would silently corrupt the stored command.
		if ch == '\x00' {
			return fmt.Errorf("inline script contains a null byte at position %d; null characters are not allowed", i)
		}
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	schedule := args[0]
	programArgs := args[1:]

	command := programArgs[0]
	if !filepath.IsAbs(command) {
		// A single non-absolute argument is treated as an inline shell script
		// and automatically wrapped in /bin/sh -c. This enables multi-line
		// scripts passed as a single quoted argument.
		if len(programArgs) != 1 {
			return fmt.Errorf("command must be an absolute path: %q\n"+
				"tip: to run a shell script inline, pass it as a single argument:\n"+
				"  ldcron add %q 'cmd1 && cmd2'", command, schedule)
		}
		script := programArgs[0]
		if err := validateInlineScript(script); err != nil {
			return err
		}
		programArgs = []string{"/bin/sh", "-c", script}
		command = "/bin/sh"
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
