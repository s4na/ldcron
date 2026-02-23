package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"os"

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

	jobs, err := job.List(agentsDir)
	if err != nil {
		return fmt.Errorf("ジョブ一覧の取得に失敗: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("登録済みのジョブはありません。")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSCHEDULE\tCOMMAND")
	fmt.Fprintln(w, "--------\t---------\t-------")
	for _, j := range jobs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", j.ID, j.Schedule, strings.Join(j.Args, " "))
	}
	return w.Flush()
}
