<div align="center">

# latch

**A task runner that stops and asks before the step you can't undo.**

Define multi-step command pipelines in YAML, put an approval gate in front of anything irreversible,<br>
and review the earlier output before it runs. Optionally trigger pipelines on a cron schedule.

[![Test](https://github.com/jackchuka/latch/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/latch/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/jackchuka/latch?sort=semver)](https://github.com/jackchuka/latch/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jackchuka/latch)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<img src="assets/demo.gif" alt="latch running a pipeline that pauses before publish, then resuming it after approval" width="800">

[Installation](#installation) · [Quick Start](#quick-start) · [Commands](#commands) · [Task Configuration](#task-configuration) · [Scheduling](#scheduling)

</div>

---

Automating a pipeline usually means choosing between running everything unattended or babysitting every step. **latch** runs the safe steps on its own, saves the pipeline state when it reaches a step marked `approve: true`, and waits. You look at what the earlier steps produced, then approve to resume or reject to drop it.

It runs any command, so it works the same for shell scripts, deploy tools, or AI agents such as `claude -p`. There is no daemon: each run is a short-lived background process, and paused pipelines are plain JSON files.

## Features

- **Approval gates** — `approve: true` pauses the pipeline _before_ that step runs
- **Output passing** — reference earlier step output in args with `{{.step_name.output}}`
- **Review queue** — list, inspect, approve, reject, or rerun paused pipelines from the CLI
- **Web dashboard** — `latch serve` to review queue items, edit a step's output before approving, and edit or run tasks in the browser
- **Cron schedules** — tasks with a `schedule` are registered with launchd on macOS
- **Agent-agnostic** — every step is just a command and its args
- **Plain files** — task definitions are YAML, queue items are JSON; `-o json` on read commands for scripting

## Installation

### Homebrew

```bash
brew install jackchuka/tap/latch
```

### Go

```bash
go install github.com/jackchuka/latch@latest
```

### Binary

Download a prebuilt binary for macOS, Linux, or Windows from the [releases page](https://github.com/jackchuka/latch/releases).

### From Source

```bash
git clone https://github.com/jackchuka/latch.git
cd latch
go build -o latch .
```

## Quick Start

1. **Write a task file:**

   ```yaml
   # deploy-review.yaml
   name: deploy-review

   steps:
     - name: check
       command: bash
       args: ["-c", "cd myapp && go test ./..."]

     - name: build
       command: bash
       args: ["-c", "cd myapp && go build -o bin/myapp ."]

     - name: deploy
       command: bash
       args: ["-c", "scp myapp/bin/myapp server:/opt/myapp/"]
       approve: true
   ```

2. **Add it and run it.** `task run` starts the pipeline in the background and prints its queue ID:

   ```bash
   latch task add -f deploy-review.yaml
   latch task run deploy-review
   ```

3. **Review and approve.** `check` and `build` run; because `deploy` has `approve: true`, the pipeline pauses before it:

   ```bash
   latch queue list            # pipelines waiting for approval
   latch queue show <id>       # output of the steps that already ran
   latch queue approve <id>    # resume: runs deploy in the background
   latch queue reject <id>     # or drop it
   ```

## Commands

### Tasks

| Command                    | Description                                                        |
| -------------------------- | ------------------------------------------------------------------ |
| `latch task add -f <file>` | Add or replace a task (and register its schedule, if set)          |
| `latch task list`          | List tasks with their schedule, whether it is installed, and steps |
| `latch task run <name>`    | Run a task now in the background                                   |
| `latch task remove <name>` | Remove a task, its queue items, and its scheduler entry            |

`task add` stores the file as `<name>.yaml`, using the `name` field inside the file rather than the file name.

### Queue

| Command                                  | Description                                                                                     |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `latch queue list`                       | List pending approvals                                                                          |
| `latch queue list --status <statuses>`   | List items by status: comma-separated statuses, or `all`                                        |
| `latch queue show <id>`                  | Show an item's status, the step it stopped at, and each completed step's output in order        |
| `latch queue approve <id>`               | Approve a pending item and resume the pipeline in the background                                |
| `latch queue reject <id>`                | Mark a pending item as rejected                                                                 |
| `latch queue rerun <id> [--from <step>]` | Start a new run from a step, reusing earlier outputs (default: the step it paused or failed at) |
| `latch queue clear`                      | Delete `done`, `failed`, and `rejected` items                                                   |
| `latch queue clear --all`                | Delete every item, including pending and running ones                                           |

```bash
latch queue list --status failed          # runs that need a rerun
latch queue list --status done,rejected   # finished runs
latch queue list --status all -o json     # everything, for scripting
```

Steps are shown by name with their position in the pipeline, for example `deploy (3 of 3)`.

### Schedule

| Command                            | Description                                                  |
| ---------------------------------- | ------------------------------------------------------------ |
| `latch schedule install`           | Register every task that has a `schedule` with the scheduler |
| `latch schedule uninstall`         | Unregister them                                              |
| `latch schedule uninstall --purge` | Unregister them and delete all queue items                   |

### Dashboard and Other

| Command                          | Description                                                   |
| -------------------------------- | ------------------------------------------------------------- |
| `latch serve [-p <port>]`        | Start the web dashboard in the background (default port 8080) |
| `latch serve --foreground`       | Run the dashboard in the foreground                           |
| `latch serve --stop [-p <port>]` | Stop the dashboard on one port, or every running dashboard    |
| `latch version`                  | Print version information                                     |

`task list`, `queue list`, `queue show`, and `version` accept `-o json` for machine-readable output.

## How It Works

A **task** is a named list of **steps**. Each step runs one command and captures its combined stdout and stderr. A step with `approve: true` pauses the pipeline before it runs, so you can review what came before.

```
latch task run deploy-review
  ↓
  check   → runs, output captured
  build   → runs, output captured
  deploy  → approve: true → PAUSED (deploy has not run)
  ↓
  saved to the queue as "pending"
  ↓
latch queue approve <id>
  ↓
  deploy  → runs → DONE
```

A queue item is `running`, `pending`, `done`, `failed`, or `rejected`. If a step exits non-zero, the item is marked `failed` and the error is stored in its JSON file; `latch queue list --status failed` lists them, and `latch queue rerun <id>` tries again from that step.

## Task Configuration

```yaml
name: my-task # required; used as the task's ID
schedule: "0 9 * * 1-5" # optional; cron expression (macOS only, see Scheduling)
timeout: 300 # optional; seconds per step (default 300)

steps:
  - name: step_one # letters, digits, underscores; unique within the task
    command: echo
    args: ["hello"]

  - name: step_two
    command: bash
    args: ["-c", "echo got: {{.step_one.output}}"]

  - name: step_three
    command: bash
    args: ["-c", "echo final: {{.step_two.output}}"]
    approve: true # pause for approval before running this step
```

- `args` are Go templates. `{{.step_name.output}}` is the earlier step's output with the trailing newline removed. Templates are not applied to `command`.
- Commands run directly, not through a shell. Use `bash -c` when you need pipes, `cd`, or environment variable expansion.
- This file is also in the repository as [`example.yml`](example.yml).

Agent-specific flags go in `args`, not in latch config:

```yaml
- name: generate
  command: claude
  args: ["-p", "Summarize the open issues", "--allowedTools", "Read"]
```

To get notified when something is waiting, add a regular step right before the gate:

```yaml
- name: notify
  command: bash
  args: ["-c", "curl -s -d 'deploy ready for review' $NTFY_URL"]

- name: deploy
  command: bash
  args: ["-c", "scp myapp/bin/myapp server:/opt/myapp/"]
  approve: true
```

## Scheduling

Add a `schedule` field and `latch task add` registers the task with launchd, writing `~/Library/LaunchAgents/com.latch.<name>.plist`. At each trigger, launchd runs `latch task run <name>`; the pipeline then pauses at its gate as usual.

```yaml
# daily-standup.yaml
name: daily-standup
schedule: "0 9 * * 1-5" # 09:00, Monday to Friday

steps:
  - name: gather
    command: claude
    args: ["-p", "Gather my activity from yesterday"]

  - name: draft
    command: claude
    args: ["-p", "Write a standup report:\n{{.gather.output}}"]

  - name: post
    command: claude
    args:
      [
        "-p",
        "Post to #standup:\n{{.draft.output}}",
        "--dangerously-skip-permissions",
      ]
    approve: true
```

- Scheduling is supported on macOS only. On other platforms, `task add` rejects a task that has one; everything else works.
- The cron expression has the standard 5 fields (minute, hour, day of month, month, day of week):

  | Syntax  | Example          | Meaning                                       |
  | ------- | ---------------- | --------------------------------------------- |
  | `*`     | `0 9 * * *`      | every value: daily at 9:00                    |
  | `*/N`   | `*/15 * * * *`   | every N: every 15 minutes                     |
  | `a-b`   | `0 9 * * 1-5`    | a range: weekdays at 9:00                     |
  | `a-b/N` | `0 8-18/2 * * *` | every N within a range: 8:00, 10:00, …, 18:00 |
  | `a,b,c` | `0 9,17 * * *`   | a list: 9:00 and 17:00                        |

  Day of week is 0–7, where both 0 and 7 are Sunday. Names such as `MON` or `JAN` are not supported.

- As in cron, when both day of month and day of week are restricted (`0 9 1 * 1`), the task runs on either. launchd cannot express cron's other case, a step like `*/2` in one day field combined with a restricted value in the other, so `task add` and `schedule install` report an error for it.
- The plist captures your `PATH` at the time of `task add` / `schedule install`, so commands resolve the same way they do in your shell.
- launchd coalesces missed triggers: if the Mac is asleep at trigger time, the task runs once on wake. Nothing irreversible happens without approval, so a late run is harmless.

## Storage

latch follows the XDG Base Directory specification (`$XDG_CONFIG_HOME` and `$XDG_DATA_HOME`):

```
~/.config/latch/
  tasks/
    <name>.yaml        # task definitions

~/.local/share/latch/
  queue/
    <id>.json          # queue items: status, step outputs, errors
  serve-<port>.pid     # running dashboards
```

Queue files are plain JSON, so `cat` and `jq` work on them.

## Development

```bash
go test ./...
go build -o latch .
```

Regenerate the demo GIF with `vhs assets/demo.tape`.

## License

[MIT](LICENSE)
