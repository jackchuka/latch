---
name: latch-schedule
description: >
  Interactively create a latch task YAML and register it as a scheduled pipeline.
  Walks the user through naming, steps, approval gates, and cron schedule.
  Use when the user wants to schedule a new task, create a pipeline, or says
  "schedule a task", "add a latch task", "create a pipeline", "/latch-schedule".
---

# Schedule a Latch Task

Interactively author a latch task YAML and register it with the system scheduler.

## When to Use

- User wants to schedule a new task or pipeline
- User says "schedule a task", "add a latch task", "create a pipeline"
- User invokes `/latch-schedule`

## Prerequisites

- `latch` CLI installed and in PATH (`which latch`)

## Arguments

Parse from the user's invocation message:

- **Task description**: Any prose about what the task should do
- **Schedule hint**: e.g., "every morning", "weekdays at 9", "hourly"
- If no arguments, start from scratch with questions

## Workflow

### Phase 1: Verify Prerequisites

1. Run `which latch` to confirm the CLI is available.
2. If not found, tell the user: "latch CLI not found. Install with `go install github.com/jackchuka/latch@latest`."

### Phase 2: Gather Task Details

Ask these one at a time. If the user provided a description in their invocation, infer as much as possible and confirm rather than asking from scratch.

**2a. Task name**

Ask: "What should we call this task? (kebab-case, e.g., `daily-standup`, `weekly-report`)"

Validate: must be kebab-case, no spaces, no special characters beyond hyphens.

**2b. Steps**

Ask: "What should this task do? Describe the steps — I'll turn them into a pipeline."

For each step the user describes, determine:
- `name`: short kebab-case identifier
- `command`: the executable to run
- `args`: array of arguments
- `approve`: whether this step requires human approval before it runs

If the user's description is vague, suggest concrete steps and confirm. For example, if they say "gather data and post to Slack", propose:
```yaml
steps:
  - name: gather
    command: claude
    args: ["-p", "Gather the data"]
  - name: post
    command: claude
    args: ["-p", "Post to Slack: {{.gather.output}}"]
    approve: true
```

Remind the user that steps can reference earlier step outputs with `{{.step-name.output}}`.

**2c. Approval gates**

For each step, ask: "Should this step pause for your approval before continuing? (Recommended before irreversible actions like posting, sending, or deleting)"

Default suggestion: put `approve: true` on the irreversible step itself. The pipeline pauses *before* running that step, letting you review prior outputs first.

**2d. Schedule**

Ask: "How often should this run?"

Offer presets:
- **Daily at 9 AM** → `0 9 * * *`
- **Weekdays at 9 AM** → `0 9 * * 1-5`
- **Every hour** → `0 * * * *`
- **Custom cron** → let user type a 5-field cron expression

If the user gave a schedule hint in their invocation (e.g., "every morning"), map it to the closest preset and confirm.

### Phase 3: Generate and Confirm YAML

1. Assemble the complete task YAML:

```yaml
name: <task-name>
schedule: "<cron expression>"

steps:
  - name: <safe-step>
    command: <cmd>
    args: [<args>]
  - name: <irreversible-step>
    command: <cmd>
    args: [<args>]
    approve: true          # require approval before running this step
```

2. Present the YAML to the user and ask: "Does this look right? (yes / edit / cancel)"
3. If **edit**: ask what to change, revise, re-confirm.
4. If **cancel**: stop.

### Phase 4: Write and Register

1. Write the YAML to a temp file:
```bash
TMPFILE=$(mktemp /tmp/latch-XXXXXX.yaml)
cat > "$TMPFILE" << 'EOF'
<the yaml content>
EOF
```

2. Register with latch:
```bash
latch task add -f "$TMPFILE"
```

3. Verify registration:
```bash
latch task list
```

4. Clean up the temp file:
```bash
rm "$TMPFILE"
```

5. Report success: "Task `<name>` scheduled (`<cron description>`). It will run automatically. Use `latch queue list` to review pending approvals."

## Error Handling

| Error | Action |
|-------|--------|
| `latch` not in PATH | Tell user to install, stop |
| Invalid cron expression | `latch add` will reject it — show the error, ask user to fix |
| Task name already exists | `latch task add` overwrites — warn user before proceeding |
| YAML write fails | Show error, output the YAML so user can save manually |

## Tips

- Cron format: `minute hour day month weekday` (5 fields, no seconds)
- Weekdays: 0=Sunday, 1=Monday, ..., 6=Saturday
- Use `*` for "every" — e.g., `0 * * * *` = every hour at minute 0
- Steps run sequentially; each step's stdout is available to later steps via `{{.step-name.output}}`
- `approve: true` pauses the pipeline *before* running that step and saves state to the queue for human review
- Cron schedules use the system's local timezone

## Examples

**Example 1: Minimal invocation (ask all questions)**
```
User: "schedule a task"
Action: Ask for name → ask what it should do → ask about approval gates → ask for schedule → generate YAML → confirm → register
```

**Example 2: Rich invocation (infer and confirm)**
```
User: "schedule a daily standup that gathers updates from Slack and posts a summary, weekdays at 9"
Action: Infer name=daily-standup, 2 steps (gather + post), approve on post, schedule=0 9 * * 1-5 → present pre-filled YAML → confirm → register
```

**Example 3: Explicit name and schedule**
```
User: "/latch-schedule weekly-report every Monday at 8am"
Action: Use name=weekly-report, schedule=0 8 * * 1 → ask what the task should do → generate YAML → confirm → register
```
