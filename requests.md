# 要件定義書: ldcron

## Context / Goal

macOSユーザーがcron形式の慣れ親しんだ記法でlaunchdジョブを管理できるCLIツール。  
内部的にはlaunchdをそのまま使用し、plistの生成・配置・登録・削除をラップする。

---

## In Scope

- `add` / `list` / `remove` / `run` コマンド
- cron5フィールド（分・時・日・月・曜日）のパースと変換
- `~/Library/LaunchAgents` への plist 管理
- `gui/$(id -u)` ドメインでの launchctl 操作
- Homebrew（tap形式）での配布
- ログ出力（stdout/stderr を `~/Library/Logs/ldcron/<id>.log` へ）

## Out of Scope

- 秒フィールド対応
- `system` ドメイン対応（v2以降）
- `edit` コマンド（v2以降）
- TUI / GUI
- cronファイル一括インポート
- 失敗時リトライ
- シェルラップ（引数は配列形式のみ）

---

## User Stories

1. **追加**  
   macOSユーザーとして、cron式とコマンドパスを指定してジョブを登録したい。launchdのplist知識なしに使えるようにするため。

2. **一覧確認**  
   macOSユーザーとして、登録済みジョブをcron式・コマンドとともに一覧表示したい。現在の登録状況を把握するため。

3. **削除**  
   macOSユーザーとして、IDを指定してジョブを削除したい。不要なジョブをlaunchdから完全に除去するため。

4. **手動実行**  
   macOSユーザーとして、IDを指定してジョブを即時実行したい。動作確認やデバッグのため。

5. **重複検知**  
   macOSユーザーとして、同一のschedule+commandを再登録しようとしたとき、すでに登録済みである旨のメッセージを受け取りたい。意図しない状態変更を防ぐため。

6. **ログ確認の導線**  
   macOSユーザーとして、ジョブの出力ログがどこに保存されるか把握したい。トラブル時に自力調査できるようにするため。

7. **不正入力の検知**  
   macOSユーザーとして、不正なcron式や相対パスを指定したとき即座にエラーを受け取りたい。ジョブが意図せず登録されることを防ぐため。

---

## Acceptance Criteria（Gherkin）

### 主要シナリオ

```gherkin
Feature: ジョブ追加

  Scenario: 正常なcron式とコマンドでジョブを追加する
    Given ユーザーが `ldcron add "0 12 * * *" /usr/local/bin/myscript.sh` を実行する
    When cron式が有効でコマンドが絶対パスである
    Then plistが ~/Library/LaunchAgents/com.ldcron.<id>.plist に生成される
    And launchctl bootstrap が実行される
    And 割り当てられたIDが表示される

Feature: ジョブ一覧

  Scenario: 登録済みジョブを一覧表示する
    Given ~/Library/LaunchAgents に com.ldcron.* のplistが存在する
    When ユーザーが `ldcron list` を実行する
    Then ID・元のcron式・コマンドパスが表形式で表示される

Feature: ジョブ削除

  Scenario: IDを指定してジョブを削除する
    Given 指定IDのジョブが登録済みである
    When ユーザーが `ldcron remove <id>` を実行する
    Then launchctl bootout が実行される
    And 対応するplistファイルが削除される
```

### 例外シナリオ

```gherkin
Feature: 重複登録の防止

  Scenario: 同一schedule+commandを再登録しようとする
    Given 同一のschedule+commandがすでに登録されている（ハッシュが一致）
    When ユーザーが `ldcron add` で同じ内容を実行する
    Then 「すでに登録済みです（ID: <id>）」というメッセージが表示される
    And plistの上書きは行われない
    And exit code 1 で終了する

Feature: 不正入力の検知

  Scenario: 不正なcron式を指定する
    Given ユーザーが `ldcron add "99 25 * * *" /usr/local/bin/script.sh` を実行する
    When cron式のパースが失敗する
    Then エラーメッセージが表示される
    And plistは生成されない
    And exit code 1 で終了する
```

---

## NFR（非機能要件）

| 区分 | 要件 |
|------|------|
| 性能 | 全コマンドの実行時間 < 100ms（launchctl呼び出しを除く） |
| バイナリ | サイズ < 10MB、外部依存なし（単一バイナリ） |
| セキュリティ | コマンドは絶対パス必須。シェルラップ禁止。引数は配列形式 |
| 対応OS | macOS 12以降 |
| ドメイン | `gui/$(id -u)`（ユーザードメイン固定） |
| ログ | `~/Library/Logs/ldcron/<id>.log` にstdout/stderrを記録 |
| 配布 | Homebrew tap形式でインストール可能 |

---

## Open Questions

1. **`*/5 * * * *` の変換精度**  
   `StartInterval=300` に変換した場合、launchd はマシン起動からの相対時間で動作する（cron のように :00/:05 に揃わない）。この挙動差をユーザーに通知するか？

2. **`run` コマンドの終了待ち**  
   `launchctl kickstart -k` はジョブ完了を待たない。ユーザーに結果（exit code）を返す必要があるか？

3. **plist内のメタ情報**  
   元のcron式をplist内に保存する方法（コメント or カスタムキー）は実装時に決定。ただし `list` コマンドで cron 式を復元できることが必須。

4. **コマンドに引数を含む場合の対応**  
   例: `ldcron add "0 12 * * *" /usr/bin/ruby /path/to/script.rb` のような複数トークンの扱いをCLI仕様として確定する必要がある。
