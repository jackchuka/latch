package detach

import (
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
)

func TestApproveNonExistent(t *testing.T) {
	q := queue.New(t.TempDir())

	_, err := Approve(q, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent item")
	}
}

func TestRunSavesItem(t *testing.T) {
	q := queue.New(t.TempDir())

	item := &queue.Item{
		ID:             "run-001",
		Task:           "deploy",
		Status:         queue.StatusRunning,
		StepsCompleted: map[string]pipeline.StepResult{},
	}

	// Run will spawn a process that fails (test binary doesn't have "queue exec"),
	// but the item should be persisted regardless.
	_, _ = Run(q, item)

	loaded, err := q.Load("run-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Task != "deploy" {
		t.Errorf("Task: got %q, want %q", loaded.Task, "deploy")
	}
}

func TestApproveNonPending(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running", queue.StatusRunning},
		{"done", queue.StatusDone},
		{"failed", queue.StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := queue.New(t.TempDir())
			item := &queue.Item{
				ID:             "item-1",
				Task:           "deploy",
				Created:        time.Now(),
				Status:         tt.status,
				StepsCompleted: map[string]pipeline.StepResult{},
			}
			if err := q.Save(item); err != nil {
				t.Fatalf("Save: %v", err)
			}

			_, err := Approve(q, "item-1")
			if err == nil {
				t.Fatalf("expected error for %s item", tt.status)
			}
		})
	}
}
