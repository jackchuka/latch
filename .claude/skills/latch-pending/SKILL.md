---
name: latch-pending
description: >
  Review and batch-process all pending latch approval items.
  Shows pending pipeline outputs, lets you approve or reject multiple items at once.
  Use when the user wants to review pending work, check the queue, or says
  "check pending", "review queue", "what's waiting", "approve tasks", "/latch-pending".
---

# Review Pending Latch Items

Batch review all pending approval items — show outputs, then approve or reject in bulk.

## When to Use

- User wants to review pending approvals
- User says "check pending", "review queue", "what's waiting for approval"
- User invokes `/latch-pending`

## Prerequisites

- `latch` CLI installed and in PATH (`which latch`)

## Arguments

Parse from the user's invocation message:

- **Action hint**: e.g., "approve all", "reject item 2" -> pre-fill selections in Phase 5 (still confirm before executing)
- **Item filter**: Any task name mentioned -> only show matching items
- If no arguments, show all pending items and ask interactively

## Workflow

### Phase 1: Verify Prerequisites

1. Run `which latch` to confirm the CLI is available.
2. If not found, tell the user: "latch CLI not found. Install with `go install github.com/jackchuka/latch@latest`."

### Phase 2: Fetch Pending Items

1. Run `latch queue list` to list all pending items.
2. Parse the output table. Each row has: ID, TASK, CREATED, PAUSED AT.
3. If no pending items, report: "All clear — nothing pending." and stop.

### Phase 3: Fetch Details for Each Item

For each pending item, run `latch queue show <id>` to get:
- Task name
- Created timestamp
- Paused at step number
- Completed step outputs (the `--- step: <name> ---` sections)

Run these in parallel where possible.

### Phase 4: Present Summary

Present all pending items in a numbered table with their step outputs:

```
## Pending Items

### 1. daily-standup (20260228-093045-daily-standup)
Created: 2026-02-28 09:30 | Paused at step: 2

**Step: gather**
> [output from the gather step, first 200 chars if over 500]

**Step: draft**
> [output from the draft step]

---

### 2. weekly-report (20260228-100000-weekly-report)
Created: 2026-02-28 10:00 | Paused at step: 1

**Step: collect**
> [output]

---
```

If a step's output exceeds 500 characters, show the first 200 characters with "... (truncated)" and offer to show the full output on request.

### Phase 5: Batch Action

Ask the user to select items for approval or rejection. Use a multi-select question:

"Which items do you want to **approve**? (e.g., '1, 2' or 'all' or 'none')"

Then: "Which items do you want to **reject**? (from the remaining items, e.g., '3' or 'none')"

Items not selected for either action are skipped (left pending).

### Phase 6: Execute Actions

For each approved item:
```bash
latch queue approve <id>
```

For each rejected item:
```bash
latch queue reject <id>
```

Report results after each action. If an action fails, show the error and continue with remaining items.

After all actions are executed, run `latch queue list` again to check if any approved items re-appeared (paused at a new approval gate).

### Phase 7: Summary

```
## Done

- Approved: N item(s)
- Rejected: M item(s)
- Skipped: K item(s) (still pending)
```

If any items were approved and their pipelines completed, note: "Approved pipelines have resumed and completed."

If any approved pipelines paused again at another approval gate, note: "Some pipelines paused at a new approval gate. Run `/latch-pending` again to review."

## Error Handling

| Error | Action |
|-------|--------|
| `latch` not in PATH | Tell user to install, stop |
| No pending items | Report "all clear", stop |
| `latch queue show` fails for an item | Warn, skip that item, continue with others |
| `latch queue approve` fails | Show error, continue with remaining items |
| `latch queue reject` fails | Show error, continue with remaining items |

## Tips

- Pending items are JSON files in `~/.local/share/latch/queue/` — inspectable with `cat` or `jq`
- Each item contains the full output of completed steps, so you can review exactly what happened
- Approving an item resumes the pipeline from where it paused
- Rejecting marks it as rejected — the remaining steps will not run

## Examples

**Example 1: Default review**
```
User: "check pending"
Action: latch queue list -> latch queue show for each -> present summary -> ask approve/reject -> execute -> summary
```

**Example 2: Direct approval intent**
```
User: "approve all pending tasks"
Action: latch queue list -> latch queue show for each -> present summary -> pre-select "all" for approval, confirm with user -> execute -> summary
```

**Example 3: Nothing pending**
```
User: "/latch-pending"
Action: latch queue list -> no items -> "All clear — nothing pending."
```
