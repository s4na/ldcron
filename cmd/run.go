package cmd

import (
	"fmt"
	"strings"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/spf13/cobra"
)

var runForce bool

var runCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "Run a job immediately by ID",
	Long: `Trigger an immediate run of a job via launchd (asynchronous).

The job runs in the background. Check the log for results:
  tail -f ~/Library/Logs/ldcron/<id>.log

With --force, any currently running instance is killed before restarting.
Omit --force unless you need to forcefully restart the job.

Examples:
  ldcron run abc12345
  ldcron run --force abc12345`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runRun,
}

func init() {
	runCmd.Flags().BoolVar(&runForce, "force", false, "kill any running instance before starting")
}

func runRun(_ *cobra.Command, args []string) error {
	id := args[0]

	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}

	j, err := job.Find(agentsDir, id)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}
	if j == nil {
		return fmt.Errorf("job not found: %s", id)
	}

	// Ensure the log directory exists before kickstarting so the process can
	// write its output immediately on launch (mirrors the order in runAdd).
	var logD string
	if j.Managed {
		logD, err = logDir()
		if err != nil {
			return fmt.Errorf("failed to get log directory: %w", err)
		}
	}

	lc, err := launchctl.New()
	if err != nil {
		return fmt.Errorf("failed to initialize launchctl client: %w", err)
	}
	if err = lc.Kickstart(j.Label, runForce); err != nil {
		return fmt.Errorf("failed to run job: %w", err)
	}

	fmt.Printf("Job started in background\n")
	fmt.Printf("  ID:      %s\n", j.ID)
	fmt.Printf("  Command: %s\n", strings.Join(j.Args, " "))
	if j.Managed {
		fmt.Printf("  Log:     %s/%s.log\n", logD, j.ID)
		fmt.Printf("\nFollow log output:\n  tail -f %s/%s.log\n", logD, j.ID)
	}
	return nil
}
