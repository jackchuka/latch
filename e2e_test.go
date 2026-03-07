package main_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
)

func TestEndToEnd(t *testing.T) {
	// 1. Setup — create temp dirs for tasks and queue
	taskDir := t.TempDir()
	queueDir := t.TempDir()

	// 2. Write a task YAML with 2 steps:
	//    - generate: echo "hello from step 1"
	//    - deliver: echo "delivering: {{.generate.output}}" with approve: true
	taskYAML := `name: e2e-test
schedule: "0 9 * * *"
steps:
  - name: generate
    command: echo
    args:
      - "hello from step 1"
  - name: deliver
    command: echo
    args:
      - "delivering: {{.generate.output}}"
    approve: true
`
	taskFile := filepath.Join(taskDir, "e2e-test.yaml")
	if err := os.WriteFile(taskFile, []byte(taskYAML), 0o644); err != nil {
		t.Fatalf("write task YAML: %v", err)
	}

	// 3. Load task from the YAML file
	tk, err := task.Load(taskFile)
	if err != nil {
		t.Fatalf("task.Load: %v", err)
	}
	if tk.Name != "e2e-test" {
		t.Fatalf("task name: got %q, want %q", tk.Name, "e2e-test")
	}
	if len(tk.Steps) != 2 {
		t.Fatalf("task steps: got %d, want 2", len(tk.Steps))
	}

	// 4. Run pipeline from step 0 — should pause before deliver (approve: true)
	q := queue.New(queueDir)
	result, err := pipeline.RunWithContext(tk, 0, 0, nil)
	if err != nil {
		t.Fatalf("pipeline.RunWithContext: %v", err)
	}

	// 5. Verify paused — result.Status == pipeline.StatusPaused
	if result.Status != pipeline.StatusPaused {
		t.Fatalf("expected status %q, got %q", pipeline.StatusPaused, result.Status)
	}

	// 6. Save queue item and verify
	item := &queue.Item{
		ID:             "e2e-test-001",
		Task:           tk.Name,
		PausedAtStep:   result.PausedAtStep,
		StepsCompleted: result.StepsCompleted,
		Status:         queue.StatusPending,
	}
	if err := q.Save(item); err != nil {
		t.Fatalf("Save queue item: %v", err)
	}

	pending, err := q.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending items: got %d, want 1", len(pending))
	}

	// 7. Verify draft content — the generate step output should be "hello from step 1"
	genResult, ok := item.StepsCompleted["generate"]
	if !ok {
		t.Fatal("missing step 'generate' in StepsCompleted")
	}
	if genResult.Output != "hello from step 1" {
		t.Fatalf("generate output: got %q, want %q", genResult.Output, "hello from step 1")
	}

	// 8. Approve (resume) — call RunWithContext with prior steps from the queue item
	result, err = pipeline.RunWithContext(tk, item.PausedAtStep, 0, item.StepsCompleted)
	if err != nil {
		t.Fatalf("pipeline.RunWithContext (resume): %v", err)
	}

	// 10. Verify completed — result.Status == pipeline.StatusCompleted
	if result.Status != pipeline.StatusCompleted {
		t.Fatalf("expected status %q after resume, got %q", pipeline.StatusCompleted, result.Status)
	}

	// 11. Verify template substitution — deliver step output should be
	//     "delivering: hello from step 1"
	deliverResult, ok := result.StepsCompleted["deliver"]
	if !ok {
		t.Fatal("missing step 'deliver' in StepsCompleted after resume")
	}
	expectedOutput := "delivering: hello from step 1"
	if deliverResult.Output != expectedOutput {
		t.Fatalf("deliver output: got %q, want %q", deliverResult.Output, expectedOutput)
	}

	// 12. Mark done
	item.Status = queue.StatusDone
	if err := q.Save(item); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 13. Verify empty queue — ListPending returns 0
	pending, err = q.ListPending()
	if err != nil {
		t.Fatalf("ListPending after done: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending items after done: got %d, want 0", len(pending))
	}
}
