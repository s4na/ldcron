package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:          "log",
	Short:        "Manage job log files",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var logSetupRotationCmd = &cobra.Command{
	Use:   "setup-rotation",
	Short: "Print newsyslog configuration for log rotation",
	Long: `Print a newsyslog(8) configuration snippet that rotates ldcron log files.

The generated configuration uses a glob pattern to cover all ldcron log files
under ~/Library/Logs/ldcron/. newsyslog is a macOS system service that runs
every hour and rotates log files according to /etc/newsyslog.d/*.conf.

To install:
  ldcron log setup-rotation | sudo tee /etc/newsyslog.d/com.ldcron.conf

The configuration rotates logs when they exceed 1 MB, keeps 3 compressed
archives, and requires no process signaling.`,
	SilenceUsage: true,
	RunE:         runLogSetupRotation,
}

func init() {
	logCmd.AddCommand(logSetupRotationCmd)
}

func runLogSetupRotation(cmd *cobra.Command, _ []string) error {
	dir, err := logDirPath()
	if err != nil {
		return err
	}
	logPattern := filepath.Join(dir, "*.log")

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "# ldcron log rotation — managed by ldcron\n")
	_, _ = fmt.Fprintf(w, "# See newsyslog.conf(5) for format details.\n")
	_, _ = fmt.Fprintf(w, "# logfilename\t\t\t\t\t\tmode\tcount\tsize\twhen\tflags\n")
	// Flags: G=glob pattern, N=no signal to any process, B=no rotation message in log
	_, _ = fmt.Fprintf(w, "%s\t644\t3\t1024\t*\tGNB\n", logPattern)
	return nil
}
