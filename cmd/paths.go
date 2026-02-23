package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func launchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func logDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
	}
	dir := filepath.Join(home, "Library", "Logs", "ldcron")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("ログディレクトリの作成に失敗: %w", err)
	}
	return dir, nil
}
