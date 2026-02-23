# ldcron

[![CI](https://github.com/s4na/ldcron/actions/workflows/ci.yml/badge.svg)](https://github.com/s4na/ldcron/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.25%2B-blue)](go.mod)
[![macOS](https://img.shields.io/badge/macOS-12%2B-lightgrey)](https://github.com/s4na/ldcron)

**cron式でmacOSのlaunchdジョブを管理するCLIツール。**

ldcronは、使い慣れたcron記法とmacOSの`launchd`エージェントシステムをつなぐ、シンプルなCLIです。plistファイルを一切書かずに、ジョブの登録・削除・一覧・実行が行えます。

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
brew tap s4na/ldcron
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
ldcron add <schedule> <command> [args...]
```

cron式を解析してlaunchd plistを生成し、エージェントを読み込みます。スケジュールとコマンドから生成した短いIDがジョブに割り当てられます。

```bash
# 毎日12:00に実行
ldcron add "0 12 * * *" /usr/local/bin/backup.sh

# 5分ごとに引数付きで実行
ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/worker.rb --verbose

# 平日の9〜17時に毎時実行
ldcron add "0 9-17 * * 1-5" /usr/local/bin/sync.sh
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

```
ID        SCHEDULE        COMMAND
--------  --------------- ----------------------------------
a1b2c3d4  0 12 * * *      /usr/local/bin/backup.sh
e5f6a7b8  */5 * * * *     /usr/bin/ruby /path/to/worker.rb
```

---

### `remove` — ジョブを削除する

```
ldcron remove <id>
```

launchdエージェントをアンロードし、対応するplistファイルを削除します。

```bash
ldcron remove a1b2c3d4
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

`launchctl kickstart`でジョブをトリガーします。実行は非同期です。出力はログファイルで確認してください。

```bash
ldcron run a1b2c3d4
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# 実行中のインスタンスを強制終了して再起動する場合
ldcron run --force a1b2c3d4
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

---

## ファイル配置

| 種別          | パス                                              |
|---------------|---------------------------------------------------|
| launchd plist | `~/Library/LaunchAgents/com.ldcron.<id>.plist`    |
| ジョブログ    | `~/Library/Logs/ldcron/<id>.log`                  |

---

## 注意事項

- **絶対パスが必須。** launchdはログインシェルを経由しないため`$PATH`が展開されません。`which <コマンド名>`でフルパスを確認してください。
- **シェルラップなし。** シェル組み込みコマンドやパイプを使う場合はインタープリタを明示してください: `ldcron add "* * * * *" /bin/sh -c 'echo hello >> /tmp/out.txt'`
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
相対パスやシェルエイリアスは使用できません。`which <コマンド名>`でフルパスを確認してください。

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
