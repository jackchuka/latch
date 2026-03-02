package runner

import (
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
)

const DefaultTimeout = 300 // seconds

// ResolveTimeout returns the task's timeout or the default if unset.
func ResolveTimeout(tk *task.Task) time.Duration {
	timeout := DefaultTimeout
	if tk.Timeout > 0 {
		timeout = tk.Timeout
	}
	return time.Duration(timeout) * time.Second
}

// SaveResult creates a new queue item from a pipeline result and persists it.
func SaveResult(q *queue.Queue, taskName string, result *pipeline.Result, runErr error) *queue.Item {
	item := queue.NewItem(taskName, result.Status, result.StepsCompleted, result.PausedAtStep)
	if runErr != nil {
		item.Error = runErr.Error()
	}
	return item
}

// ApplyResult updates an existing queue item with pipeline results.
func ApplyResult(item *queue.Item, result *pipeline.Result, runErr error) {
	switch result.Status {
	case pipeline.StatusCompleted:
		item.Status = queue.StatusDone
	case pipeline.StatusPaused:
		item.Status = queue.StatusPending
	case pipeline.StatusFailed:
		item.Status = queue.StatusFailed
		if runErr != nil {
			item.Error = runErr.Error()
		}
	}
	item.PausedAtStep = result.PausedAtStep
	item.MergeSteps(result.StepsCompleted)
}
