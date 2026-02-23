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
	Short: "IDを指定してジョブを即時実行する",
	Long: `指定したIDのジョブをlaunchdに即時実行させます（非同期）。

実行はバックグラウンドで行われます。結果はログで確認してください:
  tail -f ~/Library/Logs/ldcron/<id>.log

--force を指定すると、実行中のインスタンスを強制終了してから再起動します。
強制終了が必要なければ --force は付けないでください。

例:
  ldcron run abc12345
  ldcron run --force abc12345`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runRun,
}

func init() {
	runCmd.Flags().BoolVar(&runForce, "force", false, "実行中のインスタンスを強制終了してから起動する")
}

func runRun(_ *cobra.Command, args []string) error {
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

	lc, err := launchctl.New()
	if err != nil {
		return fmt.Errorf("launchctlクライアントの初期化に失敗: %w", err)
	}
	if err := lc.Kickstart(j.Label, runForce); err != nil {
		return fmt.Errorf("ジョブの実行に失敗: %w", err)
	}

	fmt.Printf("ジョブをバックグラウンドで起動しました\n")
	fmt.Printf("  ID:      %s\n", j.ID)
	fmt.Printf("  コマンド: %s\n", strings.Join(j.Args, " "))
	fmt.Printf("  ログ:    ~/Library/Logs/ldcron/%s.log\n", j.ID)
	fmt.Printf("\nログをリアルタイムで確認:\n  tail -f ~/Library/Logs/ldcron/%s.log\n", j.ID)
	return nil
}
