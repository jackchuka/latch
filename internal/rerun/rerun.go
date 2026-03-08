package rerun

import (
	"fmt"
	"strings"

	"github.com/jackchuka/latch/internal/detach"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
)

// Result contains the new queue item and PID from a rerun.
type Result struct {
	Item *queue.Item
	PID  int
}

// Run creates a new queue item from an existing one and spawns background execution.
// If fromStep is empty, it defaults to the original's paused/failed step.
func Run(q *queue.Queue, original *queue.Item, tk *task.Task, fromStep string) (*Result, error) {
	if original.Status == queue.StatusRunning {
		return nil, fmt.Errorf("item %s cannot be rerun (status: %s)", original.ID, original.Status)
	}

	var stepIdx int
	var stepName string
	if fromStep != "" {
		stepIdx = tk.StepIndex(fromStep)
		if stepIdx < 0 {
			names := make([]string, len(tk.Steps))
			for i, s := range tk.Steps {
				names[i] = s.Name
			}
			return nil, fmt.Errorf("step %q not found; valid steps: %s", fromStep, strings.Join(names, ", "))
		}
		stepName = fromStep
	} else {
		stepIdx = original.PausedAtStep
		if stepIdx >= 0 && stepIdx < len(tk.Steps) {
			stepName = tk.Steps[stepIdx].Name
		}
	}

	prior := make(map[string]pipeline.StepResult)
	for i := 0; i < stepIdx; i++ {
		name := tk.Steps[i].Name
		if sr, ok := original.StepsCompleted[name]; ok {
			prior[name] = sr
		}
	}

	item := queue.NewItem(original.Task, pipeline.StatusPaused, prior, stepIdx)
	item.Status = queue.StatusRunning
	item.RerunFrom = original.ID
	item.RerunFromStep = stepName

	pid, err := detach.Run(q, item)
	if err != nil {
		return nil, err
	}

	return &Result{Item: item, PID: pid}, nil
}
