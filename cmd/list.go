package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/s4na/ldcron/internal/job"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "登録済みジョブを一覧表示する",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         runList,
}

func runList(_ *cobra.Command, _ []string) error {
	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}

	jobs, warnings, err := job.List(agentsDir)
	if err != nil {
		return fmt.Errorf("ジョブ一覧の取得に失敗: %w", err)
	}

	if len(jobs) == 0 && len(warnings) == 0 {
		fmt.Println("登録済みのジョブはありません。")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSCHEDULE\tCOMMAND")
	_, _ = fmt.Fprintln(w, "--------\t---------\t-------")
	for _, j := range jobs {
		schedule := j.Schedule
		if schedule == "" {
			schedule = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", j.ID, schedule, strings.Join(j.Args, " "))
	}
	for _, warn := range warnings {
		_, _ = fmt.Fprintf(w, "[WARNING]\t-\t%s: %v\n", warn.Path, warn.Err)
	}
	return w.Flush()
}
