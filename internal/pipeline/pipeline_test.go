package pipeline

import (
	"strings"
	"testing"

	"github.com/jackchuka/latch/internal/task"
)

func TestRunAllStepsNoApproval(t *testing.T) {
	tk := &task.Task{
		Name: "simple",
		Steps: []task.Step{
			{Name: "step1", Command: "echo", Args: []string{"hello"}},
			{Name: "step2", Command: "echo", Args: []string{"world"}},
		},
	}

	result, err := Run(tk, 0, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Errorf("Status: got %q, want %q", result.Status, StatusCompleted)
	}
	if len(result.StepsCompleted) != 2 {
		t.Errorf("StepsCompleted: got %d, want 2", len(result.StepsCompleted))
	}
	if out := result.StepsCompleted["step1"].Output; out != "hello" {
		t.Errorf("step1 output: got %q, want %q", out, "hello")
	}
	if out := result.StepsCompleted["step2"].Output; out != "world" {
		t.Errorf("step2 output: got %q, want %q", out, "world")
	}
}

func TestRunPausesAtApproval(t *testing.T) {
	tk := &task.Task{
		Name: "with-approval",
		Steps: []task.Step{
			{Name: "gather", Command: "echo", Args: []string{"data"}},
			{Name: "draft", Command: "echo", Args: []string{"message"}, Approve: true},
			{Name: "post", Command: "echo", Args: []string{"sent"}},
		},
	}

	result, err := Run(tk, 0, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != StatusPaused {
		t.Errorf("Status: got %q, want %q", result.Status, StatusPaused)
	}
	if len(result.StepsCompleted) != 1 {
		t.Errorf("StepsCompleted: got %d, want 1", len(result.StepsCompleted))
	}
	if _, ok := result.StepsCompleted["gather"]; !ok {
		t.Error("missing step 'gather' in completed")
	}
	if _, ok := result.StepsCompleted["draft"]; ok {
		t.Error("step 'draft' should not be in completed (gate is pre-execution)")
	}
	if result.PausedAtStep != 1 {
		t.Errorf("PausedAtStep: got %d, want 1", result.PausedAtStep)
	}
}

func TestRunResumesFromStep(t *testing.T) {
	tk := &task.Task{
		Name: "with-approval",
		Steps: []task.Step{
			{Name: "gather", Command: "echo", Args: []string{"data"}},
			{Name: "draft", Command: "echo", Args: []string{"message"}, Approve: true},
			{Name: "post", Command: "echo", Args: []string{"sent"}},
		},
	}

	// First run: pauses before draft (approve: true)
	result1, err := Run(tk, 0, 0)
	if err != nil {
		t.Fatalf("Run (first): %v", err)
	}
	if result1.Status != StatusPaused {
		t.Fatalf("first run Status: got %q, want %q", result1.Status, StatusPaused)
	}

	// Resume from step 1 (draft) — approval clears the gate
	result2, err := Run(tk, 1, 0)
	if err != nil {
		t.Fatalf("Run (resume): %v", err)
	}
	if result2.Status != StatusCompleted {
		t.Errorf("resume Status: got %q, want %q", result2.Status, StatusCompleted)
	}
	if _, ok := result2.StepsCompleted["draft"]; !ok {
		t.Error("missing step 'draft' in completed after resume")
	}
	if _, ok := result2.StepsCompleted["post"]; !ok {
		t.Error("missing step 'post' in completed after resume")
	}
}

func TestRunWithContextResumesWithPriorSteps(t *testing.T) {
	tk := &task.Task{
		Name: "context-resume",
		Steps: []task.Step{
			{Name: "gather", Command: "echo", Args: []string{"data"}},
			{Name: "draft", Command: "echo", Args: []string{"prior: {{.gather.output}}"}, Approve: true},
			{Name: "post", Command: "echo", Args: []string{"sent"}},
		},
	}

	// Prior context: only gather completed before the gate
	prior := map[string]StepResult{
		"gather": {Output: "data", Duration: "1ms"},
	}

	// Resume at step 1 (draft) — approval clears the gate
	result, err := RunWithContext(tk, 1, 0, prior)
	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Errorf("Status: got %q, want %q", result.Status, StatusCompleted)
	}
	if out := result.StepsCompleted["draft"].Output; out != "prior: data" {
		t.Errorf("draft output: got %q, want %q", out, "prior: data")
	}
	if _, ok := result.StepsCompleted["gather"]; !ok {
		t.Error("missing prior step 'gather' in completed")
	}
}

func TestTemplateSubstitution(t *testing.T) {
	tk := &task.Task{
		Name: "template-test",
		Steps: []task.Step{
			{Name: "step1", Command: "echo", Args: []string{"hello"}},
			{Name: "step2", Command: "echo", Args: []string{"got: {{.step1.output}}"}},
		},
	}

	result, err := Run(tk, 0, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Errorf("Status: got %q, want %q", result.Status, StatusCompleted)
	}

	out := result.StepsCompleted["step2"].Output
	if !strings.Contains(out, "got: hello") {
		t.Errorf("step2 output: got %q, want it to contain %q", out, "got: hello")
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name    string
		steps   []task.Step
		wantErr string
	}{
		{
			name: "command not found",
			steps: []task.Step{
				{Name: "bad", Command: "nonexistent_cmd_xyz", Args: []string{"arg"}},
			},
			wantErr: "execute step",
		},
		{
			name: "template parse error in args",
			steps: []task.Step{
				{Name: "bad", Command: "echo", Args: []string{"{{.bad"}},
			},
			wantErr: "resolve args",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &task.Task{Name: "err_test", Steps: tt.steps}

			result, err := Run(tk, 0, 0)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
			if result.Status != StatusFailed {
				t.Errorf("Status: got %q, want %q", result.Status, StatusFailed)
			}
		})
	}
}
