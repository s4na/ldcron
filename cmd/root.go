// Package cmd implements the ldcron CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.19"

var rootCmd = &cobra.Command{
	Use:          "ldcron",
	Version:      version,
	Short:        "macOS launchd job manager with cron syntax",
	SilenceUsage: true,
	// Show help when invoked without subcommands.
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(runCmd)

	// Save the default subcommand help, then set a custom help for the root command.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(), rootHelpText())
	})
}

func rootHelpText() string {
	return fmt.Sprintf(`ldcron — macOS launchd job scheduler  v%s

  A CLI tool for managing launchd jobs on macOS using cron syntax.
  Automates plist generation, installation, registration, and removal.

Getting started:

  1. Register a job
       ldcron add "0 12 * * *" /usr/local/bin/backup.sh
       → Schedules backup.sh to run every day at 12:00

  2. List registered jobs
       ldcron list
       → Shows all jobs with their ID, schedule, and command

  3. Test-run a job manually
       ldcron run <id>
       → Triggers the job immediately without waiting for the schedule

  4. Remove a job
       ldcron remove <id>
       → Unregisters the job from launchd and deletes the plist file

Commands:

  add     <schedule> <command> [args...]   Register a new job
  list                                     List all registered jobs
  remove  <id>                             Delete a job by ID
  run     <id>                             Run a job immediately

Cron expression format (minute hour day month weekday):

  "0 12 * * *"     Every day at 12:00
  "*/5 * * * *"    Every 5 minutes at fixed times (:00, :05, :10 ...)
  "0 9 * * 1-5"    Weekdays (Mon–Fri) at 9:00
  "30 8 1 * *"     1st of every month at 8:30

Flags:

  -h, --help      Show this help message
  -v, --version   Show version information

More: ldcron <command> --help
`, version)
}
