package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a job by ID",
	Long: `Unregister a job from launchd and delete its plist file.

With --force, the plist is deleted even if launchctl bootout fails.
Use this to recover when launchd and the plist are out of sync.

Examples:
  ldcron remove abc12345
  ldcron remove abc12345 --force`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "delete plist even if bootout fails")
}

func runRemove(_ *cobra.Command, args []string) error {
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

	// Unload from launchd.
	lc, err := launchctl.New()
	if err != nil {
		return fmt.Errorf("failed to initialize launchctl client: %w", err)
	}
	if bootoutErr := lc.Bootout(j.Label); bootoutErr != nil {
		if !removeForce {
			return fmt.Errorf("failed to remove from launchctl: %w\nuse --force to force deletion", bootoutErr)
		}
		fmt.Fprintf(os.Stderr, "warning: bootout failed (continuing with --force): %v\n", bootoutErr)
	}

	// Remove plist file.
	backupPath, err := job.Remove(agentsDir, j)
	if err != nil {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	if backupPath != "" {
		fmt.Printf("External job plist renamed (not deleted)\n")
		fmt.Printf("  ID:      %s\n", j.ID)
		fmt.Printf("  Command: %s\n", strings.Join(j.Args, " "))
		fmt.Printf("  Backup:  %s\n", backupPath)
	} else {
		fmt.Printf("Job removed\n")
		fmt.Printf("  ID:       %s\n", j.ID)
		fmt.Printf("  Schedule: %s\n", j.Schedule)
		fmt.Printf("  Command:  %s\n", strings.Join(j.Args, " "))
	}
	return nil
}
