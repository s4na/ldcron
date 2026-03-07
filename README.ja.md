# ldcron

[![CI](https://github.com/s4na/ldcron/actions/workflows/ci.yml/badge.svg)](https://github.com/s4na/ldcron/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.25%2B-blue)](go.mod)
[![macOS](https://img.shields.io/badge/macOS-12%2B-lightgrey)](https://github.com/s4na/ldcron)

**cron式でmacOSのlaunchdジョブを管理するCLIツール。**

ldcronは、使い慣れたcron記法とmacOSの`launchd`エージェントシステムをつなぐ、シンプルなCLIです。plistファイルを一切書かずに、ジョブの登録・削除・一覧・実行が行えます。

ldcronは**ネイティブのlaunchdと完全互換**です：
- ldcronで登録したジョブは、標準のlaunchd plistファイルとして保存されます。ldcronを使わなくなっても、登録済みのジョブはそのまま動作し続けます。実行時にldcronバイナリへの依存はありません。
- `ldcron list`・`ldcron remove`・`ldcron run`は、ldcronで作成したジョブだけでなく、`~/Library/LaunchAgents/`にある**すべてのplist**を操作できます。既存のlaunchdエージェントもldcronで管理できます。

[English README](README.md)

---

## なぜldcronか？

macOSは推奨ジョブスケジューラとして`cron`を`launchd`に置き換えました。しかしlaunchdを使うには、冗長なXML plistファイル、特定のディレクトリ配置、手動での`launchctl`実行が必要で、スクリプトをスケジュール実行するだけの作業にしては手間がかかります。

ldcronはその手間をすべて肩代わりします。cron式を渡せば、ldcronがplistの生成・エージェントの読み込み・ジョブのライフサイクル管理をすべて行います。

```bash
# ldcron導入前 — XMLを書いて ~/Library/LaunchAgents/ に置いて launchctl load して…
# ldcron導入後:
ldcron add "0 12 * * *" /usr/local/bin/backup.sh
```

---

## インストール

### Homebrew（推奨）

```bash
brew tap s4na/ldcron https://github.com/s4na/ldcron
brew install ldcron
```

### go install

```bash
go install github.com/s4na/ldcron@latest
```

**動作要件:** macOS 12 (Monterey) 以降

---

## クイックスタート

```bash
# 毎日12時にスクリプトをスケジュール
ldcron add "0 12 * * *" /usr/local/bin/backup.sh

# 登録済みジョブを一覧表示
ldcron list

# ジョブを即時実行（動作確認に）
ldcron run a1b2c3d4

# ログをリアルタイムで確認
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# ジョブを削除
ldcron remove a1b2c3d4
```

---

## コマンドリファレンス

### `add` — ジョブを登録する

```
ldcron add <schedule> <command|script> [args...]
```

cron式を解析してlaunchd plistを生成し、エージェントを読み込みます。スケジュールとコマンドから生成した短いIDがジョブに割り当てられます。

コマンドは2つの方法で指定できます：

- **絶対パス** — バイナリのフルパスと任意の引数を直接渡します。
- **インラインシェルスクリプト** — 引数を1つだけ渡します。ldcron が自動的に `/bin/sh -c "..."` でラップします。`$'...'` 記法を使うと複数行スクリプトも指定できます。

```bash
# 毎日12:00に実行（絶対パス）
ldcron add "0 12 * * *" /usr/local/bin/backup.sh

# 5分ごとに引数付きで実行
ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/worker.rb --verbose

# 平日の9〜17時に毎時実行
ldcron add "0 9-17 * * 1-5" /usr/local/bin/sync.sh

# インライン1行シェルコマンド
ldcron add "0 * * * *" "echo hello && date >> /tmp/log.txt"

# インライン複数行シェルスクリプト（$'...' で \n が実際の改行になる）
ldcron add "0 * * * *" $'cd /tmp\nfind . -name "*.log" -mtime +30 -delete\necho cleaned'
```

```
ジョブを追加しました
  ID:       a1b2c3d4
  スケジュール: 0 12 * * *
  コマンド:   /usr/local/bin/backup.sh
  ログ:      ~/Library/Logs/ldcron/a1b2c3d4.log
```

> **補足:** 同一のスケジュール＋コマンドの重複登録は防止されます。同じ入力からは常に同じIDが生成されます。

---

### `list` — 登録済みジョブを一覧表示する

```
ldcron list
```

`~/Library/LaunchAgents/`にある**すべての**plistを表示します（ldcron以外で作成したジョブも含む）。外部ジョブはlaunchdラベル全体がIDとして表示され、cron式が保存されていない場合はスケジュール欄に`-`が表示されます。

```
ID                        SCHEDULE        COMMAND
----------------          --------------- ----------------------------------
a1b2c3d4e5f6a7b8          0 12 * * *      /usr/local/bin/backup.sh
e5f6a7b8a1b2c3d4          */5 * * * *     /usr/bin/ruby /path/to/worker.rb
com.apple.ccachefixer     -               /usr/libexec/ccachefixer
```

---

### `remove` — ジョブを削除する

```
ldcron remove <id>
```

launchdエージェントをアンロードし、対応するplistファイルを削除します。ldcron管理ジョブは短いhex IDで、外部ジョブはlaunchdラベル全体で指定します。

```bash
ldcron remove a1b2c3d4e5f6a7b8
ldcron remove com.apple.ccachefixer
```

```
ジョブを削除しました
  ID:       a1b2c3d4
  スケジュール: 0 12 * * *
  コマンド:   /usr/local/bin/backup.sh
```

---

### `run` — ジョブを即時実行する

```
ldcron run [--force] <id>
```

`launchctl kickstart`でジョブをトリガーします。実行は非同期です。ldcron管理ジョブはログパスが表示されます。外部ジョブのログはそのplistの`StandardOutPath`設定を参照してください。

```bash
ldcron run a1b2c3d4e5f6a7b8
tail -f ~/Library/Logs/ldcron/a1b2c3d4e5f6a7b8.log

# 外部ジョブを即時実行する
ldcron run com.apple.ccachefixer

# 実行中のインスタンスを強制終了して再起動する場合
ldcron run --force a1b2c3d4e5f6a7b8
```

```
ジョブをバックグラウンドで起動しました
  ID:      a1b2c3d4
  コマンド: /usr/local/bin/backup.sh
  ログ:    ~/Library/Logs/ldcron/a1b2c3d4.log
```

> **補足:** `--force`なしでは、すでに実行中のジョブはエラーを返します。`--force`は実行中のインスタンスを強制終了してから再起動します。実行中のジョブを中断してよい場合のみ使用してください。

---

## cron式の構文

5フィールド（**分 時 日 月 曜日**）の標準形式を使用します。

```
┌──────────── 分 (0–59)
│ ┌────────── 時 (0–23)
│ │ ┌──────── 日 (1–31)
│ │ │ ┌────── 月 (1–12)
│ │ │ │ ┌──── 曜日 (0=日 … 6=土, 7=日)
│ │ │ │ │
* * * * *
```

| 構文           | 例                    | 説明                             |
|----------------|-----------------------|----------------------------------|
| `*`            | `* * * * *`           | 毎分                             |
| 固定値         | `0 12 * * *`          | 毎日12:00                        |
| ステップ       | `*/15 * * * *`        | 15分ごと                         |
| 範囲           | `0 9-17 * * *`        | 9〜17時の毎時0分                 |
| リスト         | `0 9,12,18 * * *`     | 9時・12時・18時                  |
| 範囲＋ステップ  | `0-30/10 * * * *`     | 0・10・20・30分                  |
| 曜日指定       | `0 9 * * 1-5`         | 月〜金の9:00                     |
| `@hourly`      | `@hourly`             | `0 * * * *` と同じ              |
| `@daily`       | `@daily`              | `0 0 * * *` と同じ              |
| `@weekly`      | `@weekly`             | `0 0 * * 0` と同じ              |
| `@monthly`     | `@monthly`            | `0 0 1 * *` と同じ              |
| `@yearly`      | `@yearly`             | `0 0 1 1 *` と同じ              |

### よく使うパターン

```bash
"* * * * *"        # 毎分
"*/5 * * * *"      # 5分ごと（:00, :05, :10 … という絶対時刻でトリガー）
"0 0 * * *"        # 毎日深夜0時
"@daily"           # 上と同じ
"0 9 * * 1-5"      # 平日の9:00
"30 8 1 * *"       # 毎月1日の8:30
```

---

## ログの確認

各ジョブのstdoutとstderrは `~/Library/Logs/ldcron/<id>.log` に記録されます。

```bash
# リアルタイムで確認
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# 最後の100行を表示
tail -n 100 ~/Library/Logs/ldcron/a1b2c3d4.log
```

### ログローテーション

ログファイルはデフォルトでは無限に増え続けます。ldcronはnewsyslog(8)の設定を生成するコマンドを提供しており、すべてのldcronログファイルを自動的にローテーションできます。

```bash
# newsyslog設定を生成・インストール（初回のみ、sudo必要）
ldcron log setup-rotation | sudo tee /etc/newsyslog.d/com.ldcron.conf
```

生成される設定は、各ログファイルが1MBを超えた時点でローテーションし、gzip圧縮したアーカイブを3世代保持します。プロセスへのシグナル送信は不要です（launchdはジョブ実行ごとにログファイルを開き直すため）。newsyslogはシステムのlaunchdジョブにより毎時自動実行されるため、追加のスケジュール設定は不要です。

---

## ファイル配置

| 種別          | パス                                              |
|---------------|---------------------------------------------------|
| launchd plist | `~/Library/LaunchAgents/com.ldcron.<id>.plist`    |
| ジョブログ    | `~/Library/Logs/ldcron/<id>.log`                  |

---

## 注意事項

- **複数引数コマンドは絶対パスが必須。** launchdはログインシェルを経由しないため`$PATH`が展開されません。`which <コマンド名>`でフルパスを確認してください。あるいはインラインスクリプト（上記 `add` 参照）を使うと ldcron が `/bin/sh -c` でラップします。
- **シェル組み込みコマンドやパイプ** を使う場合はシェルを明示してください。絶対パスで `/bin/sh -c '...'` と書くか、1引数のインラインスクリプトとして渡してください。
- **インラインスクリプトの改行は LF（Unix形式）を使用してください。** Windows形式の改行（CRLF）が含まれる場合、CR文字がそのまま plist に保存され `/bin/sh` に渡されます。多くのシェルは問題なく処理しますが、予期しない動作が起きた場合は改行を LF に変換してから渡してください（例: `tr -d '\r'`）。
- **`run`は非同期。** ジョブの完了を待ちません。結果はログで確認してください。
- **`run --force`は実行中プロセスを強制終了します。** `--force`なしでは、実行中のジョブはエラーを返します。`--force`は実行中のインスタンスを即座に終了してから再起動します。
- **ステップ式は絶対時刻でトリガーされます。** `*/5 * * * *`は登録時刻から5分後ではなく、:00, :05, :10 …という絶対的な分数でトリガーされます。
- **ログインセッション限定。** ジョブは`gui/<uid>`ドメインに読み込まれ、ログイン中のみ実行されます。システムレベルや常駐タスクには適していません。

---

## トラブルシューティング

**v0.1.2以前からのアップグレード**
v0.1.3でジョブIDが8文字から16文字に変更されました。既存のジョブはそのまま動作し続けますが、同じスケジュール＋コマンドを再登録すると重複として検出されず新しいエントリが作成されます。`ldcron list`で古い8文字のIDを確認し、`ldcron remove <古いID>`でアンロードしてから再登録してください。

**`already registered`（すでに登録済みです）**
同一のスケジュール＋コマンドがすでに登録されています。`ldcron list`で既存ジョブを確認し、必要であれば`ldcron remove`で削除してから再登録してください。

**`command must be an absolute path`（コマンドは絶対パスで指定してください）**
複数引数を渡す場合、最初の引数は絶対パスでなければなりません。`which <コマンド名>`でフルパスを確認してください。または1つのインラインスクリプトとして渡す方法もあります：`ldcron add "..." 'cmd1 && cmd2'`

**`invalid cron expression`（cron式が無効）**
フィールドの値が範囲外か、5フィールド未満です。[cron式の構文](#cron式の構文)を確認してください。

---

## コントリビューション

コントリビューションを歓迎します。大きな変更を送る前に、まずIssueを開いて方向性を確認してください。

```bash
# クローンしてビルド
git clone https://github.com/s4na/ldcron.git
cd ldcron
go build ./...

# テスト実行（macOS必須）
go test -race ./...

# Lint
golangci-lint run
```

---

## ライセンス

MIT © [s4na](https://github.com/s4na)
