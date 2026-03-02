package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jackchuka/latch/internal/task"
	"github.com/jackchuka/latch/internal/tmpl"
)

type StepResult struct {
	Output   string `json:"output"`
	Duration string `json:"duration"`
}

const (
	StatusCompleted = "completed"
	StatusPaused    = "paused"
	StatusFailed    = "failed"
)

type Result struct {
	Status         string
	StepsCompleted map[string]StepResult
	PausedAtStep   int
}

// Run executes a task's steps starting from startStep with no prior context.
func Run(tk *task.Task, startStep int, timeout time.Duration) (*Result, error) {
	return RunWithContext(tk, startStep, timeout, nil)
}

// RunWithContext executes a task's steps starting from startStep, seeded with
// prior step results so that template substitution (e.g. {{.gather.output}})
// works when resuming a paused pipeline.
func RunWithContext(tk *task.Task, startStep int, timeout time.Duration, prior map[string]StepResult) (*Result, error) {
	completed := make(map[string]StepResult)

	// Build template data, starting with any prior results
	templateData := make(map[string]any)
	for name, sr := range prior {
		completed[name] = sr
		templateData[name] = map[string]any{
			"output": sr.Output,
		}
	}

	for i := startStep; i < len(tk.Steps); i++ {
		step := tk.Steps[i]

		// Gate check: pause before running a step that requires approval.
		// Skip the check for startStep — that step was already approved.
		if step.Approve && i != startStep {
			return &Result{
				Status:         StatusPaused,
				StepsCompleted: completed,
				PausedAtStep:   i,
			}, nil
		}

		// Resolve template substitutions in args
		resolvedArgs, err := resolveArgs(step.Args, templateData)
		if err != nil {
			return &Result{
				Status:         StatusFailed,
				StepsCompleted: completed,
				PausedAtStep:   i,
			}, fmt.Errorf("resolve args for step %q: %w", step.Name, err)
		}

		// Execute the command
		start := time.Now()
		output, err := execute(step.Command, resolvedArgs, timeout)
		duration := time.Since(start).Round(time.Millisecond).String()
		if err != nil {
			return &Result{
				Status:         StatusFailed,
				StepsCompleted: completed,
				PausedAtStep:   i,
			}, fmt.Errorf("execute step %q: %w", step.Name, err)
		}

		completed[step.Name] = StepResult{
			Output:   output,
			Duration: duration,
		}

		// Update template data for subsequent steps
		templateData[step.Name] = map[string]any{
			"output": output,
		}
	}

	return &Result{
		Status:         StatusCompleted,
		StepsCompleted: completed,
		PausedAtStep:   len(tk.Steps),
	}, nil
}

func resolveArgs(args []string, data map[string]any) ([]string, error) {
	resolved := make([]string, len(args))
	for i, arg := range args {
		s, err := tmpl.Resolve("arg", arg, data)
		if err != nil {
			return nil, fmt.Errorf("resolve arg %q: %w", arg, err)
		}
		resolved[i] = s
	}
	return resolved, nil
}

// execute runs a command with the given arguments and returns its combined
// output with the trailing newline trimmed. If timeout is positive, the
// command is killed after that duration.
func execute(command string, args []string, timeout time.Duration) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
