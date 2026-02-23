// Package cmd implements the ldcron CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ldcron",
	Short: "cron形式でlaunchdジョブを管理するCLIツール",
	Long: `ldcron - macOS launchd job manager with cron syntax

cron式とコマンドパスを指定してlaunchdジョブを管理します。
内部的にはlaunchdをそのまま使用し、plistの生成・配置・登録・削除をラップします。

例:
  ldcron add "0 12 * * *" /usr/local/bin/myscript.sh
  ldcron list
  ldcron remove <id>
  ldcron run <id>`,
	SilenceUsage: true,
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
}
