# latch

Task runner with approval gates. Define multi-step command pipelines, gate irreversible actions behind human approval, and optionally schedule them with cron.

Agent-agnostic — runs any command, not just AI agents. No daemon to maintain.

## Install

```bash
go install github.com/jackchuka/latch@latest
```

Or build from source:

```bash
git clone https://github.com/jackchuka/latch.git
cd latch
go build -o latch .
```

## Quick Start

Create a task file:

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
    args: ["-c", "scp bin/myapp server:/opt/myapp/"]
    approve: true
```

Add and run it:

```bash
latch task add -f deploy-review.yaml
latch task run deploy-review
```

The pipeline runs `check`, then `build`. Because `deploy` has `approve: true`, latch pauses before running it and waits for you:

```bash
latch queue list               # see what's waiting
latch queue show <id>          # review the output
latch queue approve <id>       # resume — deploys
```

### Adding a Schedule

Tasks can optionally run on a cron schedule. Add a `schedule` field:

```yaml
# daily-standup.yaml
name: daily-standup
schedule: "0 9 * * 1-5"

steps:
  - name: gather
    command: claude
    args: ["-p", "Gather my activity from GitHub, Slack, and Calendar"]

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

```bash
latch task add -f daily-standup.yaml
```

This registers with launchd automatically. At 9 AM on weekdays, latch runs the pipeline, pauses before the `post` step, and waits for you.

## Commands

### Tasks

| Command                    | Description                                               |
| -------------------------- | --------------------------------------------------------- |
| `latch task add -f <file>` | Add a task (registers with scheduler if schedule is set)  |
| `latch task remove <name>` | Remove a task and its data (queue items, scheduler entry) |
| `latch task list`          | List all tasks                                            |
| `latch task run <name>`    | Run a task now                                            |

### Queue

| Command                    | Description                                   |
| -------------------------- | --------------------------------------------- |
| `latch queue list`         | List pending approvals                        |
| `latch queue show <id>`    | Show a queued item                            |
| `latch queue approve <id>` | Approve a queued item and resume the pipeline |
| `latch queue reject <id>`  | Reject and discard a queued item              |
| `latch queue clear`        | Clear finished (done) queue items             |
| `latch queue clear --all`  | Clear all queue items including pending       |

### Schedule

| Command                            | Description                                        |
| ---------------------------------- | -------------------------------------------------- |
| `latch schedule install`           | Register scheduled tasks with the system scheduler |
| `latch schedule uninstall`         | Unregister all tasks from the system scheduler     |
| `latch schedule uninstall --purge` | Unregister and delete all queue items              |

### Other

| Command         | Description               |
| --------------- | ------------------------- |
| `latch version` | Print version information |

## How It Works

A **task** is a named pipeline of **steps**. Each step runs a command and captures stdout. Any step can have `approve: true`, which pauses the pipeline _before_ running that step so a human can review prior outputs.

```
latch task run <name>
  ↓
  step 1: check     → runs, captures output
  step 2: build     → runs, captures output
  step 3: deploy    → approve: true → PAUSED (deploy has not run yet)
  ↓
  saved to queue
  ↓
  latch queue approve <id>
  ↓
  step 3: deploy    → runs with {{.build.output}} substituted → DONE
```

Steps can reference earlier outputs using `{{.step_name.output}}` in their args.

Commands that display data (`task list`, `queue list`, `queue show`, `version`) accept `-o json` for machine-readable output.

## Task Configuration

```yaml
name: my-task
schedule: "0 9 * * 1-5" # optional, cron expression (minute hour day month weekday)
timeout: 300 # optional, seconds

steps:
  - name: step_one
    command: echo
    args: ["hello"]

  - name: step_two
    command: bash
    args: ["-c", "echo got: {{.step_one.output}}"]

  - name: step_three
    command: bash
    args: ["-c", "echo final: {{.step_two.output}}"]
    approve: true # require approval before running this step
```

Steps are agent-agnostic. Agent-specific flags go in `args`, not in latch config:

```yaml
- name: generate
  command: claude
  args: ["-p", "Do something", "--allowedTools", "Read,Write"]
```

Need notifications before an approval gate? Add a notification step — no special config needed:

```yaml
- name: notify
  command: bash
  args: ["-c", "curl -s -d 'deploy ready for review' $NTFY_URL"]

- name: deploy
  command: bash
  args: ["-c", "scp bin/myapp server:/opt/myapp/"]
  approve: true
```

## Storage

Follows the XDG Base Directory specification:

```
~/.config/latch/
  tasks/
    daily-standup.yaml     # task definitions

~/.local/share/latch/
  queue/
    <id>.json              # paused pipeline state
```

Queue files are plain JSON — inspectable with `cat` or `jq`.

## Scheduling

When a task has a `schedule` field, `latch task add` registers it with the system scheduler (launchd on macOS). Scheduling supports macOS currently, with an interface designed for future backends (systemd, cron, etc.).

launchd coalesces missed triggers. If your Mac is asleep at trigger time, the task runs once on wake. This is fine because the approval gate means nothing irreversible happens automatically.

## License

MIT
