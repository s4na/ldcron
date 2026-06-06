package cmd

import (
	"fmt"
	"os"

	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/s4na/ldcron/internal/migrate"
	"github.com/spf13/cobra"
)

var migrateDryRun bool
var migrateQuiet bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy ldcron plist files to current IDs",
	Long: `Migrate ldcron-managed plist files created by older releases to the
current deterministic ID format.

Loaded jobs are moved from the legacy launchd label to the current label.
Unloaded jobs are rewritten on disk without being loaded.

Examples:
  ldcron migrate
  ldcron migrate --dry-run`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show migrations without changing plist files")
	migrateCmd.Flags().BoolVarP(&migrateQuiet, "quiet", "q", false, "suppress output when no migration is needed")
}

func runMigrate(_ *cobra.Command, _ []string) error {
	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}
	logD, err := logDir()
	if err != nil {
		return err
	}
	lc, err := launchctl.New()
	if err != nil {
		return fmt.Errorf("failed to initialize launchctl client: %w", err)
	}

	results, warnings, err := migrate.Run(agentsDir, logD, lc, migrate.Options{DryRun: migrateDryRun})
	for _, warn := range warnings {
		fmt.Fprintf(os.Stderr, "warning: skipped unreadable plist %s: %v\n", warn.Path, warn.Err)
	}
	if err != nil {
		return err
	}

	if len(results) == 0 {
		if !migrateQuiet {
			fmt.Println("No legacy ldcron jobs to migrate.")
		}
		return nil
	}

	if migrateDryRun {
		fmt.Println("Legacy ldcron jobs that would be migrated:")
	} else {
		fmt.Println("Migrated legacy ldcron jobs:")
	}
	for _, result := range results {
		action := "migrated"
		if result.Consolidated {
			action = "consolidated"
		}
		fmt.Printf("  %s -> %s (%s)\n", result.OldID, result.NewID, action)
	}
	return nil
}
