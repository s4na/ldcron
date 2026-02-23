// Package cmd implements the ldcron CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.16"

var rootCmd = &cobra.Command{
	Use:          "ldcron",
	Version:      version,
	Short:        "macOS launchd job manager with cron syntax",
	SilenceUsage: true,
	// 引数なしで実行された場合はヘルプを表示する。
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

	// サブコマンドのデフォルト help を保存してから、root 用カスタム help を設定する。
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}
		fmt.Fprint(cmd.OutOrStdout(), rootHelpText())
	})
}

func rootHelpText() string {
	return fmt.Sprintf(`ldcron — macOS launchd job scheduler  v%s

  macOS の launchd を使ってスケジュールジョブを管理する CLI ツールです。
  cron 形式でスケジュールを指定し、指定時刻にコマンドを自動実行します。

基本ワークフロー:

  1. ジョブを登録する
       ldcron add "0 12 * * *" /usr/local/bin/backup.sh
       → 毎日 12:00 に backup.sh を実行するジョブを登録します

  2. 登録したジョブを確認する
       ldcron list
       → ID・スケジュール・コマンドの一覧を表示します

  3. ジョブをテスト実行する
       ldcron run <id>
       → 今すぐ手動でジョブを起動してテストできます

  4. 不要なジョブを削除する
       ldcron remove <id>
       → launchd からジョブを登録解除してplistを削除します

コマンド:

  add     <schedule> <command> [args...]   ジョブを新規登録する
  list                                     登録済みジョブを一覧表示する
  remove  <id>                             ジョブを削除する
  run     <id>                             ジョブを即時実行する

cron 式のフォーマット（分 時 日 月 曜日）:

  "0 12 * * *"     毎日 12:00 に実行
  "*/5 * * * *"    5 分おきに実行（:00, :05, :10 ... の固定時刻）
  "0 9 * * 1-5"    平日（月〜金）の 9:00 に実行
  "30 8 1 * *"     毎月 1 日の 8:30 に実行

フラグ:

  -h, --help      このヘルプを表示する
  -v, --version   バージョン情報を表示する

詳細: ldcron <command> --help
`, version)
}
