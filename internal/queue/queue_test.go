package queue

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	item := &Item{
		ID:           "test-001",
		Task:         "deploy",
		Created:      time.Now().Truncate(time.Second),
		PausedAtStep: 2,
		Status:       StatusPending,
		StepsCompleted: map[string]pipeline.StepResult{
			"build": {Output: "ok", Duration: "1.2s"},
		},
	}

	if err := q.Save(item); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := q.Load("test-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != item.ID {
		t.Errorf("ID: got %q, want %q", loaded.ID, item.ID)
	}
	if loaded.Task != item.Task {
		t.Errorf("Task: got %q, want %q", loaded.Task, item.Task)
	}
	if loaded.PausedAtStep != item.PausedAtStep {
		t.Errorf("PausedAtStep: got %d, want %d", loaded.PausedAtStep, item.PausedAtStep)
	}
	if loaded.Status != item.Status {
		t.Errorf("Status: got %q, want %q", loaded.Status, item.Status)
	}
	if !loaded.Created.Equal(item.Created) {
		t.Errorf("Created: got %v, want %v", loaded.Created, item.Created)
	}

	sr, ok := loaded.StepsCompleted["build"]
	if !ok {
		t.Fatal("StepsCompleted missing key 'build'")
	}
	if sr.Output != "ok" {
		t.Errorf("StepResult.Output: got %q, want %q", sr.Output, "ok")
	}
	if sr.Duration != "1.2s" {
		t.Errorf("StepResult.Duration: got %q, want %q", sr.Duration, "1.2s")
	}
}

func TestListPending(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	items := []*Item{
		{ID: "a", Task: "t1", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "b", Task: "t2", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "c", Task: "t3", Status: StatusDone, StepsCompleted: map[string]pipeline.StepResult{}},
	}
	for _, item := range items {
		if err := q.Save(item); err != nil {
			t.Fatalf("Save %s: %v", item.ID, err)
		}
	}

	pending, err := q.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("ListPending: got %d items, want 2", len(pending))
	}

	ids := map[string]bool{}
	for _, p := range pending {
		ids[p.ID] = true
	}
	if !ids["a"] || !ids["b"] {
		t.Errorf("ListPending: expected IDs a and b, got %v", ids)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	item := &Item{ID: "del-001", Task: "t1", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}}
	if err := q.Save(item); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := q.Delete("del-001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := q.Load("del-001")
	if err == nil {
		t.Fatal("expected error loading deleted item")
	}
}

func TestListAll(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	items := []*Item{
		{ID: "a", Task: "t1", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "b", Task: "t2", Status: StatusDone, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "c", Task: "t3", Status: StatusFailed, StepsCompleted: map[string]pipeline.StepResult{}},
	}
	for _, item := range items {
		if err := q.Save(item); err != nil {
			t.Fatalf("Save %s: %v", item.ID, err)
		}
	}

	all, err := q.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListAll: got %d items, want 3", len(all))
	}
}

func TestDeleteByStatus(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	items := []*Item{
		{ID: "a", Task: "t1", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "b", Task: "t2", Status: StatusDone, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "c", Task: "t3", Status: StatusFailed, StepsCompleted: map[string]pipeline.StepResult{}},
	}
	for _, item := range items {
		if err := q.Save(item); err != nil {
			t.Fatalf("Save %s: %v", item.ID, err)
		}
	}

	n, err := q.DeleteByStatus(StatusDone, StatusFailed)
	if err != nil {
		t.Fatalf("DeleteByStatus: %v", err)
	}
	if n != 2 {
		t.Fatalf("DeleteByStatus: deleted %d, want 2", n)
	}

	all, err := q.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 || all[0].ID != "a" {
		t.Fatalf("expected only pending item 'a', got %v", all)
	}
}

func TestDeleteByTask(t *testing.T) {
	dir := t.TempDir()
	q := New(dir)

	items := []*Item{
		{ID: "a", Task: "deploy", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "b", Task: "deploy", Status: StatusDone, StepsCompleted: map[string]pipeline.StepResult{}},
		{ID: "c", Task: "backup", Status: StatusPending, StepsCompleted: map[string]pipeline.StepResult{}},
	}
	for _, item := range items {
		if err := q.Save(item); err != nil {
			t.Fatalf("Save %s: %v", item.ID, err)
		}
	}

	n, err := q.DeleteByTask("deploy")
	if err != nil {
		t.Fatalf("DeleteByTask: %v", err)
	}
	if n != 2 {
		t.Fatalf("DeleteByTask: deleted %d, want 2", n)
	}

	all, err := q.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 || all[0].Task != "backup" {
		t.Fatalf("expected only backup item, got %v", all)
	}
}

func TestNewItem(t *testing.T) {
	tests := []struct {
		name           string
		pipelineStatus string
		wantStatus     string
	}{
		{"completed", pipeline.StatusCompleted, StatusDone},
		{"paused", pipeline.StatusPaused, StatusPending},
		{"failed", pipeline.StatusFailed, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := map[string]pipeline.StepResult{
				"build": {Output: "ok", Duration: "1s"},
			}
			item := NewItem("deploy", tt.pipelineStatus, steps, 2)

			if item.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", item.Status, tt.wantStatus)
			}
			if item.Task != "deploy" {
				t.Errorf("Task: got %q, want %q", item.Task, "deploy")
			}
			if item.PausedAtStep != 2 {
				t.Errorf("PausedAtStep: got %d, want 2", item.PausedAtStep)
			}
			if !strings.HasSuffix(item.ID, "-deploy") {
				t.Errorf("ID %q should end with -deploy", item.ID)
			}
			if item.Created.IsZero() {
				t.Error("Created should not be zero")
			}
			if item.StepsCompleted["build"].Output != "ok" {
				t.Error("StepsCompleted should contain build step")
			}
		})
	}
}

func TestMergeSteps(t *testing.T) {
	item := &Item{
		StepsCompleted: map[string]pipeline.StepResult{
			"build": {Output: "ok", Duration: "1s"},
		},
	}
	item.MergeSteps(map[string]pipeline.StepResult{
		"test":  {Output: "pass", Duration: "2s"},
		"build": {Output: "rebuilt", Duration: "3s"},
	})

	if len(item.StepsCompleted) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(item.StepsCompleted))
	}
	if item.StepsCompleted["test"].Output != "pass" {
		t.Error("expected test step to be merged")
	}
	if item.StepsCompleted["build"].Output != "rebuilt" {
		t.Error("expected build step to be overwritten")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "queue")
	q := New(dir)

	item := &Item{
		ID:             "dir-test",
		Task:           "t1",
		Status:         StatusPending,
		StepsCompleted: map[string]pipeline.StepResult{},
	}
	if err := q.Save(item); err != nil {
		t.Fatalf("Save into nested dir: %v", err)
	}

	loaded, err := q.Load("dir-test")
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if loaded.ID != "dir-test" {
		t.Errorf("ID: got %q, want %q", loaded.ID, "dir-test")
	}
}

func TestMergeStepsNilMap(t *testing.T) {
	item := &Item{}
	item.MergeSteps(map[string]pipeline.StepResult{
		"build": {Output: "ok", Duration: "1s"},
	})

	if item.StepsCompleted["build"].Output != "ok" {
		t.Error("expected build step on nil map")
	}
}
