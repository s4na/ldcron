package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/cron"
	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/launchctl"
	"github.com/s4na/ldcron/internal/plist"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <schedule> <command> [args...]",
	Short: "cron式とコマンドでジョブを追加する",
	Long: `cron形式のスケジュールとコマンドを指定してlaunchdジョブを登録します。

スケジュールは5フィールドのcron式（分 時 日 月 曜日）で指定します。
コマンドは絶対パスで指定する必要があります。

例:
  ldcron add "0 12 * * *" /usr/local/bin/myscript.sh
  ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/script.rb --verbose`,
	Args:         cobra.MinimumNArgs(2),
	SilenceUsage: true,
	RunE:         runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	schedule := args[0]
	programArgs := args[1:]

	// Validate: command must be absolute path.
	command := programArgs[0]
	if !filepath.IsAbs(command) {
		return fmt.Errorf("コマンドは絶対パスで指定してください: %q", command)
	}

	// Validate: command must exist.
	if _, err := os.Stat(command); err != nil {
		return fmt.Errorf("コマンドが見つかりません: %q", command)
	}

	// Validate cron expression.
	if _, err := cron.ParseSchedule(schedule); err != nil {
		return fmt.Errorf("cron式が無効: %w", err)
	}

	agentsDir, err := launchAgentsDir()
	if err != nil {
		return err
	}
	logD, err := logDir()
	if err != nil {
		return err
	}

	j := job.NewJob(schedule, programArgs)

	// Check for duplicate.
	dup, err := job.FindDuplicate(agentsDir, j)
	if err != nil {
		return fmt.Errorf("重複チェックに失敗: %w", err)
	}
	if dup != nil {
		return fmt.Errorf("すでに登録済みです（ID: %s）", dup.ID)
	}

	// Write plist.
	plistPath, err := plist.Write(agentsDir, j.Label, j.Schedule, j.Args, logD)
	if err != nil {
		return fmt.Errorf("plistの生成に失敗: %w", err)
	}

	// Register with launchd.
	lc, err := launchctl.New()
	if err != nil {
		_ = os.Remove(plistPath)
		return fmt.Errorf("launchctlクライアントの初期化に失敗: %w", err)
	}
	if err := lc.Bootstrap(plistPath); err != nil {
		_ = os.Remove(plistPath)
		return fmt.Errorf("launchctlへの登録に失敗: %w", err)
	}

	fmt.Printf("ジョブを追加しました\n")
	fmt.Printf("  ID:       %s\n", j.ID)
	fmt.Printf("  スケジュール: %s\n", j.Schedule)
	fmt.Printf("  コマンド:   %s\n", strings.Join(j.Args, " "))
	fmt.Printf("  ログ:      %s/%s.log\n", logD, j.ID)
	return nil
}
