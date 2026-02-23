package cmd

import (
	"fmt"
	"strings"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "IDを指定してジョブを削除する",
	Long: `指定したIDのジョブをlaunchdから削除しplistファイルを消去します。

例:
  ldcron remove abc12345`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runRemove,
}

func runRemove(_ *cobra.Command, args []string) error {
	id := args[0]

	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}

	j, err := job.Find(agentsDir, id)
	if err != nil {
		return fmt.Errorf("ジョブ検索に失敗: %w", err)
	}
	if j == nil {
		return fmt.Errorf("ジョブが見つかりません: %s", id)
	}

	// Unload from launchd.
	lc, err := launchctl.New()
	if err != nil {
		return fmt.Errorf("launchctlクライアントの初期化に失敗: %w", err)
	}
	if err := lc.Bootout(j.Label); err != nil {
		return fmt.Errorf("launchctlからの削除に失敗: %w", err)
	}

	// Remove plist file.
	if err := job.Remove(agentsDir, j); err != nil {
		return fmt.Errorf("plistの削除に失敗: %w", err)
	}

	fmt.Printf("ジョブを削除しました\n")
	fmt.Printf("  ID:       %s\n", j.ID)
	fmt.Printf("  スケジュール: %s\n", j.Schedule)
	fmt.Printf("  コマンド:   %s\n", strings.Join(j.Args, " "))
	return nil
}
