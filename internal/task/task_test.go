package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTask(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "standup.yaml")

	data := []byte(`name: standup
schedule: "0 9 * * 1-5"
steps:
  - name: gather
    command: fetch-updates
  - name: draft
    command: summarize
    approve: true
  - name: post
    command: send-slack
    args:
      - "--channel"
      - "#standup"
`)
	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	task, err := Load(taskPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if task.Name != "standup" {
		t.Errorf("expected name %q, got %q", "standup", task.Name)
	}
	if task.Schedule != "0 9 * * 1-5" {
		t.Errorf("expected schedule %q, got %q", "0 9 * * 1-5", task.Schedule)
	}
	if len(task.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(task.Steps))
	}
	if !task.Steps[1].Approve {
		t.Error("expected step[1].Approve to be true")
	}
}

func TestLoadTask_EmptyName(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "empty.yaml")

	data := []byte(`schedule: "0 9 * * 1-5"
steps:
  - name: step1
    command: echo
`)
	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(taskPath)
	if err == nil {
		t.Fatal("expected error for empty task name, got nil")
	}
}

func TestLoadTask_InvalidStepName(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "bad.yaml")

	data := []byte(`name: my-task
steps:
  - name: step-one
    command: echo
    args: ["hello"]
`)
	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(taskPath)
	if err == nil {
		t.Fatal("expected error for hyphenated step name, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "not a valid identifier") {
		t.Errorf("error %q does not mention invalid identifier", got)
	}
}

func TestLoadTask_NoSchedule(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "ondemand.yaml")

	data := []byte(`name: ondemand
steps:
  - name: greet
    command: echo
    args: ["hello"]
`)
	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	tk, err := Load(taskPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if tk.Name != "ondemand" {
		t.Errorf("expected name %q, got %q", "ondemand", tk.Name)
	}
	if tk.Schedule != "" {
		t.Errorf("expected empty schedule, got %q", tk.Schedule)
	}
	if len(tk.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(tk.Steps))
	}
}

func TestStepIndex(t *testing.T) {
	tk := &Task{
		Steps: []Step{
			{Name: "build"},
			{Name: "test"},
			{Name: "deploy"},
		},
	}

	tests := []struct {
		name string
		step string
		want int
	}{
		{"first", "build", 0},
		{"middle", "test", 1},
		{"last", "deploy", 2},
		{"not found", "missing", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tk.StepIndex(tt.step)
			if got != tt.want {
				t.Errorf("StepIndex(%q) = %d, want %d", tt.step, got, tt.want)
			}
		})
	}
}

func TestLoadAllTasks(t *testing.T) {
	dir := t.TempDir()

	task1 := []byte(`name: task-one
schedule: "0 9 * * *"
steps:
  - name: step1
    command: echo
`)
	task2 := []byte(`name: task-two
schedule: "0 10 * * *"
steps:
  - name: step1
    command: echo
`)
	if err := os.WriteFile(filepath.Join(dir, "task1.yaml"), task1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task2.yaml"), task2, 0644); err != nil {
		t.Fatal(err)
	}

	tasks, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}
