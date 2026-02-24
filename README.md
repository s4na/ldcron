# ldcron

[![CI](https://github.com/s4na/ldcron/actions/workflows/ci.yml/badge.svg)](https://github.com/s4na/ldcron/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.25%2B-blue)](go.mod)
[![macOS](https://img.shields.io/badge/macOS-12%2B-lightgrey)](https://github.com/s4na/ldcron)

**Schedule macOS launchd jobs using familiar cron syntax.**

[日本語 README](README.ja.md)

ldcron is a minimal CLI that bridges the gap between the cron expressions you already know and the `launchd` agent system on macOS — without ever touching a plist file.

ldcron is **fully compatible with native launchd**:
- Jobs registered through ldcron are standard launchd plist files. If you stop using ldcron, all jobs continue running exactly as before — no dependency on the ldcron binary at runtime.
- `ldcron list`, `ldcron remove`, and `ldcron run` work on **any** plist in `~/Library/LaunchAgents/`, not just the ones ldcron created. You can use ldcron to manage your existing launchd agents.

---

## Why ldcron?

macOS replaced `cron` with `launchd` as the recommended job scheduler. But launchd requires verbose XML plist files, a specific directory layout, and manual `launchctl` invocations — a significant overhead just to run a script on a schedule.

ldcron handles all of that for you. You write a cron expression; ldcron writes the plist, loads the agent, and manages the job lifecycle.

```bash
# Before ldcron — write XML, copy it to ~/Library/LaunchAgents/, then run launchctl load …
# After ldcron:
ldcron add "0 12 * * *" /usr/local/bin/backup.sh
```

---

## Installation

### Homebrew (recommended)

```bash
brew tap s4na/ldcron https://github.com/s4na/ldcron
brew install ldcron
```

### go install

```bash
go install github.com/s4na/ldcron@latest
```

**Requirements:** macOS 12 (Monterey) or later.

---

## Quick start

```bash
# Schedule a script to run every day at noon
ldcron add "0 12 * * *" /usr/local/bin/backup.sh

# List all scheduled jobs
ldcron list

# Trigger a job immediately (useful for testing)
ldcron run a1b2c3d4

# Watch the output in real time
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# Remove a job when you no longer need it
ldcron remove a1b2c3d4
```

---

## Commands

### `add` — Register a job

```
ldcron add <schedule> <command|script> [args...]
```

Parses the cron expression, generates a launchd plist, and loads the agent. A short ID derived from the schedule and command is assigned to the job.

The command can be specified in two ways:

- **Absolute path** — pass the binary path and optional arguments directly.
- **Inline shell script** — pass a single unquoted argument. ldcron wraps it in `/bin/sh -c "..."` automatically. Multi-line scripts are supported via `$'...'` quoting.

```bash
# Every day at 12:00 (absolute path)
ldcron add "0 12 * * *" /usr/local/bin/backup.sh

# Every 5 minutes with arguments
ldcron add "*/5 * * * *" /usr/bin/ruby /path/to/worker.rb --verbose

# Weekdays 9–17, on the hour
ldcron add "0 9-17 * * 1-5" /usr/local/bin/sync.sh

# Inline single-line shell command
ldcron add "0 * * * *" "echo hello && date >> /tmp/log.txt"

# Inline multi-line shell script ($'...' enables \n as actual newline)
ldcron add "0 * * * *" $'cd /tmp\nfind . -name "*.log" -mtime +30 -delete\necho cleaned'
```

```
Job added
  ID:       a1b2c3d4
  Schedule: 0 12 * * *
  Command:  /usr/local/bin/backup.sh
  Log:      ~/Library/Logs/ldcron/a1b2c3d4.log
```

> **Note:** Duplicate registrations (same schedule + command) are prevented. The same inputs always produce the same ID.

---

### `list` — List registered jobs

```
ldcron list
```

Lists **all** plist files in `~/Library/LaunchAgents/`, including jobs not created by ldcron. For external jobs, the full launchd label is shown as the ID and the schedule column shows `-` if no cron expression is stored.

```
ID                        SCHEDULE        COMMAND
----------------          --------------- ----------------------------------
a1b2c3d4e5f6a7b8          0 12 * * *      /usr/local/bin/backup.sh
e5f6a7b8a1b2c3d4          */5 * * * *     /usr/bin/ruby /path/to/worker.rb
com.apple.ccachefixer     -               /usr/libexec/ccachefixer
```

---

### `remove` — Unregister a job

```
ldcron remove <id>
```

Unloads the launchd agent and deletes the corresponding plist file. For ldcron-managed jobs use the short hex ID; for external jobs use the full launchd label.

```bash
ldcron remove a1b2c3d4e5f6a7b8
ldcron remove com.apple.ccachefixer
```

```
Job removed
  ID:       a1b2c3d4
  Schedule: 0 12 * * *
  Command:  /usr/local/bin/backup.sh
```

---

### `run` — Run a job immediately

```
ldcron run [--force] <id>
```

Triggers the job via `launchctl kickstart`. Execution is asynchronous. For ldcron-managed jobs, the log path is printed; for external jobs, consult the plist's own `StandardOutPath` configuration.

```bash
ldcron run a1b2c3d4e5f6a7b8
tail -f ~/Library/Logs/ldcron/a1b2c3d4e5f6a7b8.log

# Run an external job immediately
ldcron run com.apple.ccachefixer

# Force restart even if the job is currently running
ldcron run --force a1b2c3d4e5f6a7b8
```

```
Job started in background
  ID:      a1b2c3d4
  Command: /usr/local/bin/backup.sh
  Log:     ~/Library/Logs/ldcron/a1b2c3d4.log
```

> **Note:** Without `--force`, running a job that is already executing will return an error. `--force` kills the running instance before restarting — use it only when you intend to interrupt an in-progress run.

---

## Cron expression syntax

ldcron uses the standard 5-field cron format:

```
┌──────────── minute       (0–59)
│ ┌────────── hour         (0–23)
│ │ ┌──────── day of month (1–31)
│ │ │ ┌────── month        (1–12)
│ │ │ │ ┌──── day of week  (0=Sun … 6=Sat, 7=Sun)
│ │ │ │ │
* * * * *
```

| Syntax          | Example               | Description                        |
|-----------------|-----------------------|------------------------------------|
| `*`             | `* * * * *`           | Every minute                       |
| Fixed value     | `0 12 * * *`          | Every day at 12:00                 |
| Step            | `*/15 * * * *`        | Every 15 minutes                   |
| Range           | `0 9-17 * * *`        | Top of each hour from 9:00–17:00   |
| List            | `0 9,12,18 * * *`     | At 9:00, 12:00, and 18:00          |
| Range with step | `0-30/10 * * * *`     | Minutes 0, 10, 20, 30              |
| Day of week     | `0 9 * * 1-5`         | Weekdays at 9:00                   |
| `@hourly`       | `@hourly`             | Equivalent to `0 * * * *`         |
| `@daily`        | `@daily`              | Equivalent to `0 0 * * *`         |
| `@weekly`       | `@weekly`             | Equivalent to `0 0 * * 0`         |
| `@monthly`      | `@monthly`            | Equivalent to `0 0 1 * *`         |
| `@yearly`       | `@yearly`             | Equivalent to `0 0 1 1 *`         |

### Common patterns

```bash
"* * * * *"        # every minute
"*/5 * * * *"      # every 5 minutes (at :00, :05, :10 … not relative to start time)
"0 0 * * *"        # daily at midnight
"@daily"           # same as above
"0 9 * * 1-5"      # weekdays at 9:00
"30 8 1 * *"       # 1st of every month at 8:30
```

---

## Logs

stdout and stderr for each job are written to `~/Library/Logs/ldcron/<id>.log`.

```bash
# Stream logs in real time
tail -f ~/Library/Logs/ldcron/a1b2c3d4.log

# View the last 100 lines
tail -n 100 ~/Library/Logs/ldcron/a1b2c3d4.log
```

---

## File locations

| Artifact      | Path                                              |
|---------------|---------------------------------------------------|
| launchd plist | `~/Library/LaunchAgents/com.ldcron.<id>.plist`    |
| Job log       | `~/Library/Logs/ldcron/<id>.log`                  |

---

## Caveats

- **Absolute paths only (for multi-argument commands).** launchd does not run commands in a login shell, so `$PATH` is not expanded. Use `which <command>` to find the full path, or pass an inline script (see `add` above) to let ldcron wrap it in `/bin/sh -c`.
- **Shell built-ins and pipes** require an explicit shell. Use an absolute path with `/bin/sh -c '...'`, or pass a single inline script argument.
- **`run` is asynchronous.** ldcron does not wait for the job to finish. Check the log for results.
- **`run --force` kills running processes.** Without `--force`, starting an already-running job returns an error. `--force` terminates the running instance immediately before restarting. Use with care.
- **Step expressions use absolute clock times.** `*/5 * * * *` fires at minutes :00, :05, :10 … regardless of when the job was registered — not 5 minutes after the last run.
- **Login session only.** Jobs are loaded into the `gui/<uid>` launchd domain and run only while you are logged in. They are not suitable for system-level or headless tasks.

---

## Troubleshooting

**Upgrading from v0.1.2 or earlier**
Job IDs changed from 8 to 16 characters in v0.1.3. Existing jobs continue to run, but re-registering the same schedule and command will create a new entry instead of detecting the duplicate. Run `ldcron list` to find old 8-character IDs and `ldcron remove <old-id>` to unload them before re-adding.

**`already registered`**
The exact same schedule and command are already tracked. Run `ldcron list` to inspect existing jobs; use `ldcron remove` if you want to re-register.

**`command must be an absolute path`**
When passing multiple arguments, the first must be an absolute path. Run `which <command>` to obtain the full path. Alternatively, pass a single inline shell script: `ldcron add "..." 'cmd1 && cmd2'`.

**`invalid cron expression`**
A field value is out of range or the expression has fewer than 5 fields. Check the [syntax reference](#cron-expression-syntax) above.

---

## Contributing

Contributions are welcome. Please open an issue before submitting a significant pull request so we can align on the direction.

```bash
# Clone and build
git clone https://github.com/s4na/ldcron.git
cd ldcron
go build ./...

# Run tests (requires macOS)
go test -race ./...

# Lint
golangci-lint run
```

---

## License

MIT © [s4na](https://github.com/s4na)
