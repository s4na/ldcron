# ldcron

cron形式でmacOSのlaunchdジョブを管理するCLIツール。

launchdのplistを直接書かずに、使い慣れたcron記法でジョブの登録・削除・一覧・実行ができます。

**動作要件**: macOS 12 (Monterey) 以降

---

## インストール

```bash
brew tap s4na/ldcron
brew install ldcron
```

---

## コマンドリファレンス

### `add` — ジョブを追加する

```
ldcron add <schedule> <command> [args...]
```

cron式とコマンドを指定してlaunchdジョブを登録します。
登録後に割り当てられたIDが表示されます。

```bash
# 毎日12時に実行
ldcron add "0 12 * * *" /usr/local/bin/myscript.sh

# 5分ごとに実行（引数あり）
ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/script.rb --verbose

# 平日の9〜17時に毎時実行
ldcron add "0 9-17 * * 1-5" /usr/local/bin/hourly.sh
```

出力例:
```
ジョブを追加しました
  ID:       a1b2c3d4
  スケジュール: 0 12 * * *
  コマンド:   /usr/local/bin/myscript.sh
  ログ:      ~/Library/Logs/ldcron/a1b2c3d4.log
```

---

### `list` — 登録済みジョブを一覧表示する

```
ldcron list
```

ldcronで登録したすべてのジョブをID・スケジュール・コマンドの形式で表示します。

```bash
ldcron list
```

出力例:
```
ID        SCHEDULE       COMMAND
--------  ---------      -------
a1b2c3d4  0 12 * * *     /usr/local/bin/myscript.sh
e5f6a7b8  */5 * * * *    /usr/bin/ruby /path/to/script.rb --verbose
```

---

### `remove` — ジョブを削除する

```
ldcron remove <id>
```

指定したIDのジョブをlaunchdから削除し、plistファイルも消去します。

```bash
ldcron remove a1b2c3d4
```

出力例:
```
ジョブを削除しました
  ID:       a1b2c3d4
  スケジュール: 0 12 * * *
  コマンド:   /usr/local/bin/myscript.sh
```

---

### `run` — ジョブを即時実行する

```
ldcron run <id>
```

指定したIDのジョブをlaunchdに即時実行させます。実行はバックグラウンドで行われます。
動作確認やデバッグに使用します。

```bash
ldcron run a1b2c3d4

# 実行結果はログで確認
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log
```

出力例:
```
ジョブをバックグラウンドで起動しました
  ID:      a1b2c3d4
  コマンド: /usr/local/bin/myscript.sh
  ログ:    ~/Library/Logs/ldcron/a1b2c3d4.log
```

---

## cron式の構文

5フィールド（**分 時 日 月 曜日**）の形式です。

```
┌─── 分 (0-59)
│ ┌─── 時 (0-23)
│ │ ┌─── 日 (1-31)
│ │ │ ┌─── 月 (1-12)
│ │ │ │ ┌─── 曜日 (0=日, 1=月, ..., 6=土, 7=日)
│ │ │ │ │
* * * * *
```

| 構文       | 例          | 説明                     |
|------------|-------------|-------------------------|
| `*`        | `* * * * *` | 任意の値（毎分実行）      |
| 固定値     | `0 12 * * *`| 毎日12:00               |
| ステップ   | `*/15 * * * *` | 15分ごと              |
| 範囲       | `0 9-17 * * *` | 9〜17時の毎時0分        |
| リスト     | `0 9,12,18 * * *` | 9時・12時・18時      |
| 範囲+ステップ | `0-30/10 * * * *` | 0〜30分を10分刻み  |
| 曜日指定   | `0 9 * * 1-5` | 月〜金の9:00            |

### よく使うパターン

```bash
# 毎分
"* * * * *"

# 5分ごと
"*/5 * * * *"

# 毎日深夜0時
"0 0 * * *"

# 月曜〜金曜の9:00
"0 9 * * 1-5"

# 毎月1日の8:30
"30 8 1 * *"
```

---

## ログの確認

ジョブのstdout/stderrは `~/Library/Logs/ldcron/<id>.log` に記録されます。

```bash
# リアルタイム確認
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# 最後の50行
tail -n 50 ~/Library/Logs/ldcron/a1b2c3d4.log
```

---

## ファイル配置

| 種別 | パス |
|------|------|
| plist | `~/Library/LaunchAgents/com.ldcron.<id>.plist` |
| ログ | `~/Library/Logs/ldcron/<id>.log` |

---

## エラーと対処

**「すでに登録済みです」と表示される**
同一のスケジュール＋コマンドはIDが同じになるため、重複登録を防止しています。
既存のジョブを確認してください: `ldcron list`

**「コマンドは絶対パスで指定してください」**
コマンドには絶対パスが必要です。`which <コマンド名>` で確認できます。

**「cron式が無効」**
フィールドの値が範囲外か、構文が誤っています。5フィールド形式を確認してください。

---

## 注意事項

- コマンドは絶対パスで指定してください（`/usr/local/bin/foo` など）
- シェルラップは行いません。シェル組み込みコマンドは `/bin/sh -c '...'` のように書いてください
- `run` コマンドはバックグラウンドで非同期実行します（完了待ちなし）
- launchdのドメインは `gui/<uid>`（ログイン中のユーザー固定）
