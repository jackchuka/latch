---
name: latch-explain
description: >
  Use when the user wants to understand a registered latch task, says "explain task",
  "what does X do", "show me the pipeline", "visualize task", or invokes "/latch-explain".
---

# Explain a Latch Task

Read a registered task's YAML and present a plain-language explanation with an ASCII pipeline diagram.

## When to Use

- User wants to understand what a registered task does
- User says "explain task", "what does X do", "show me the pipeline for X"
- User invokes `/latch-explain <name>`

## Prerequisites

- `latch` CLI installed and in PATH (`which latch`)

## Workflow

### Phase 1: Verify Prerequisites

Run `which latch`. If not found: "latch CLI not found. Install with `go install github.com/jackchuka/latch@latest`." Stop.

### Phase 2: Resolve Task Name

1. If user provided a name, use it.
2. Otherwise run `latch task list`.
3. No tasks → "No tasks registered. Use `latch task add -f <file>`." Stop.
4. One task → auto-select, tell user.
5. Multiple → list them, ask which to explain.

### Phase 3: Read Task Definition

Read `~/.config/latch/tasks/<name>.yaml`. Parse: `name`, `schedule`, `timeout`, `steps` (each: `name`, `command`, `args`, `approve`).

If file missing, run `latch task list` and ask user to pick.

### Phase 4: Generate Explanation

**Summary:** One sentence describing what the task does, inferred from commands/args.

**Schedule:**
- Cron set → translate to human-readable (e.g. `0 9 * * 1-5` → "Weekdays at 9:00 AM")
- No schedule → "On-demand only (`latch task run <name>`)"

**Timeout:** Show value, or "300s (default)" if unset.

**Step walkthrough:** For each step:
- What command/args it runs
- Data dependencies via `{{.X.output}}` references
- If `approve: true`: "Pipeline pauses here for human approval"

**Data flow:** List all `{{.X.output}}` references — which step produces, which consumes.

### Phase 5: Render ASCII Diagram

Vertical pipeline with boxes, arrows, data flow annotations, and approval markers.

**Template:**

```
  ┌──────────────────┐
  │  step_one        │  echo "hello"
  └────────┬─────────┘
           │ {{.step_one.output}}
           ▼
  ┌──────────────────┐
  │  step_two        │  bash -c "echo got: ..."
  └────────┬─────────┘
           │ {{.step_two.output}}
           ▼
  ┌──────────────────┐
  │ ⏸ step_three     │  bash -c "echo final: ..."
  └──────────────────┘
           ⏸ APPROVAL GATE — pipeline pauses here
```

**Rules:**
- Box width adapts to longest step name (min 16 chars inside)
- `⏸` prefix inside box for `approve: true` steps
- Command summary (first ~40 chars) to the right of each box
- Data flow annotation `{{.X.output}}` on arrows between connected steps
- Last step has no downward arrow
- `⏸ APPROVAL GATE` label below approval steps

## Error Handling

| Error | Action |
|-------|--------|
| `latch` not in PATH | Tell user to install, stop |
| Task not found | Show available tasks, ask user to pick |
| YAML unreadable | Show file path and error, stop |

## Examples

**Explain by name:**
```
User: "explain my-task"
→ read ~/.config/latch/tasks/my-task.yaml → explanation + ASCII diagram
```

**No name, one task:**
```
User: "/latch-explain"
→ latch task list → auto-select only task → explanation + diagram
```

**No name, multiple tasks:**
```
User: "visualize a task"
→ latch task list → ask which → explanation + diagram
```
