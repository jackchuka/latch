package rerun

import (
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
)

func TestRunRejectsInvalidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running", queue.StatusRunning},
		{"pending", queue.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := queue.New(t.TempDir())
			original := &queue.Item{
				ID:             "orig-" + tt.name,
				Task:           "deploy",
				Status:         tt.status,
				Created:        time.Now(),
				StepsCompleted: map[string]pipeline.StepResult{},
			}

			tk := &task.Task{
				Name:  "deploy",
				Steps: []task.Step{{Name: "build", Command: "echo"}},
			}

			_, err := Run(q, original, tk, "")
			if err == nil {
				t.Fatalf("expected error for %s item", tt.status)
			}
		})
	}
}

func TestRunInvalidStep(t *testing.T) {
	q := queue.New(t.TempDir())
	original := &queue.Item{
		ID:             "orig-2",
		Task:           "deploy",
		Status:         queue.StatusFailed,
		Created:        time.Now(),
		StepsCompleted: map[string]pipeline.StepResult{},
	}

	tk := &task.Task{
		Name:  "deploy",
		Steps: []task.Step{{Name: "build", Command: "echo"}},
	}

	_, err := Run(q, original, tk, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid step")
	}
}
