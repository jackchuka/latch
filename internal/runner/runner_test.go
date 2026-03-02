package runner_test

import (
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/runner"
	"github.com/jackchuka/latch/internal/task"
)

func TestResolveTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		want    time.Duration
	}{
		{
			name:    "default when unset",
			timeout: 0,
			want:    300 * time.Second,
		},
		{
			name:    "custom timeout",
			timeout: 60,
			want:    60 * time.Second,
		},
		{
			name:    "negative treated as default",
			timeout: -1,
			want:    300 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &task.Task{Timeout: tt.timeout}
			got := runner.ResolveTimeout(tk)
			if got != tt.want {
				t.Errorf("ResolveTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveResult(t *testing.T) {
	q := queue.New(t.TempDir())

	steps := map[string]pipeline.StepResult{
		"step1": {Output: "hello", Duration: "10ms"},
	}
	result := &pipeline.Result{
		Status:         pipeline.StatusPaused,
		StepsCompleted: steps,
		PausedAtStep:   1,
	}

	item := runner.SaveResult(q, "my-task", result, nil)

	if item.Task != "my-task" {
		t.Errorf("Task = %q, want %q", item.Task, "my-task")
	}
	if item.Status != queue.StatusPending {
		t.Errorf("Status = %q, want %q", item.Status, queue.StatusPending)
	}
	if item.PausedAtStep != 1 {
		t.Errorf("PausedAtStep = %d, want 1", item.PausedAtStep)
	}
}

func TestApplyResult(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantStatus string
	}{
		{
			name:       "completed",
			status:     pipeline.StatusCompleted,
			wantStatus: queue.StatusDone,
		},
		{
			name:       "paused",
			status:     pipeline.StatusPaused,
			wantStatus: queue.StatusPending,
		},
		{
			name:       "failed",
			status:     pipeline.StatusFailed,
			wantStatus: queue.StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &queue.Item{Status: queue.StatusRunning}
			result := &pipeline.Result{
				Status:         tt.status,
				StepsCompleted: map[string]pipeline.StepResult{"s": {Output: "out"}},
				PausedAtStep:   0,
			}

			runner.ApplyResult(item, result, nil)

			if item.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", item.Status, tt.wantStatus)
			}
		})
	}
}
