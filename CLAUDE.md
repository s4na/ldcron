# ldcron Development Guidelines

## バージョン管理

- バージョンは `cmd/root.go` の `version` 変数で管理する
- **patch バージョンは `main` へのマージ時に CI（`auto-release.yml`）が自動でインクリメントする**
- Claude が手動でバージョンを変更する必要はない

### 手動バージョン変更が必要なケース

- **minor** (`x.Y.0`): 後方互換性のある新機能追加
- **major** (`X.0.0`): 破壊的変更（CLI インターフェース変更、設定形式変更など）

上記の場合のみ `cmd/root.go` の `version` を直接編集する。その場合、CI による自動 patch バンプはその次のマージから再開される。

### バージョン定義場所

```go
// cmd/root.go
var version = "0.1.5"
```

`rootCmd` には `Version: version` を設定すること（`--version` フラグが自動追加される）。

## ブランチ運用

- **remote の `main` ブランチを壊さない**
- push 前に `git fetch origin main` で最新状態を確認する
- ローカルと remote に差分がある場合は、`git merge origin/main` でマージしてから push する
- force push は絶対に行わない

## コミット規約

- コミット対象は **今回の作業範囲のファイルのみ** に限定する
- 作業範囲外の変更（既存の未コミット差分など）は含めない
- コミット完了後、コミットしなかったファイルが残っている場合は、それらをコミットするか確認する

## README 管理

- `README.md`（英語）と `README.ja.md`（日本語）を常に同期する
- 片方を編集したら、もう片方も同じ内容に更新する
- 両ファイルに差分がある場合は、実装コードを確認した上で正しい方に合わせて両方を更新する

## launchd との完全互換性

ldcron はネイティブの launchd と完全互換であることを設計原則とする。

### ランタイム非依存

- ldcron が生成する plist は標準の launchd plist フォーマット（`StartCalendarInterval`・`ProgramArguments` 等）を使用する
- ジョブの実行には ldcron バイナリは不要。launchd が直接コマンドを実行する
- ldcron をアンインストールしても、登録済みジョブはスケジュール通りに動作し続ける
- `X-Ldcron-Schedule` カスタムキーはメタデータ用途のみ。launchd はこれを無視する

### 既存 plist の管理

- `ldcron list` / `remove` / `run` は `~/Library/LaunchAgents/` にある **すべての plist** を対象とする（`com.ldcron.*` 以外も含む）
- ldcron 管理ジョブ: ファイル名 `com.ldcron.<id>.plist` かつ `X-Ldcron-Schedule` キーを持つ → 短い hex ID で操作
- 外部ジョブ: その他の plist → launchd ラベル全体が ID として使われる

### 新機能追加時の注意

launchd との互換性を損なう変更（plist 形式の変更、独自キーの必須化など）は破壊的変更とみなし、major バージョンをインクリメントすること。

## コーディング規約

- Go の標準スタイルに従う（`gofmt`、`golangci-lint` 通過必須）
- テストは変更に合わせて追加・修正する
- `internal/` パッケージ間の循環インポートを避ける
