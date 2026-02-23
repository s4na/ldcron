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
	Short: "IDを指定してジョブを削除する",
	Long: `指定したIDのジョブをlaunchdから削除しplistファイルを消去します。

--force を指定すると、launchctlのbootoutが失敗した場合でもplistを強制削除します。
launchdとplistの状態が乖離している場合の復旧に使用してください。

例:
  ldcron remove abc12345
  ldcron remove abc12345 --force`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "bootout失敗時もplistを強制削除する")
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
		if !removeForce {
			return fmt.Errorf("launchctlからの削除に失敗: %w\n削除を強制するには --force フラグを使用してください", err)
		}
		fmt.Fprintf(os.Stderr, "警告: bootoutに失敗しました（--forceで続行）: %v\n", err)
	}

	// Remove plist file.
	backupPath, err := job.Remove(agentsDir, j)
	if err != nil {
		return fmt.Errorf("plistの操作に失敗: %w", err)
	}

	if backupPath != "" {
		fmt.Printf("管理外ジョブのplistをリネームしました（削除はされていません）\n")
		fmt.Printf("  ID:         %s\n", j.ID)
		fmt.Printf("  コマンド:     %s\n", strings.Join(j.Args, " "))
		fmt.Printf("  バックアップ:   %s\n", backupPath)
	} else {
		fmt.Printf("ジョブを削除しました\n")
		fmt.Printf("  ID:       %s\n", j.ID)
		fmt.Printf("  スケジュール: %s\n", j.Schedule)
		fmt.Printf("  コマンド:   %s\n", strings.Join(j.Args, " "))
	}
	return nil
}
