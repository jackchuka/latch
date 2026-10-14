package queue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/task"
)

const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusDone     = "done"
	StatusFailed   = "failed"
	StatusRejected = "rejected"
)

// Statuses lists every item status in lifecycle order.
var Statuses = []string{StatusPending, StatusRunning, StatusDone, StatusFailed, StatusRejected}

type Item struct {
	ID             string                         `json:"id"`
	Task           string                         `json:"task"`
	Created        time.Time                      `json:"created"`
	PausedAtStep   int                            `json:"paused_at_step"`
	StepsCompleted map[string]pipeline.StepResult `json:"steps_completed"`
	Status         string                         `json:"status"`
	Error          string                         `json:"error,omitempty"`
	PID            int                            `json:"pid,omitempty"`
	RerunFrom      string                         `json:"rerun_from,omitempty"`
	RerunFromStep  string                         `json:"rerun_from_step,omitempty"`
}

// NewItem creates a queue Item from pipeline results.
// pipelineStatus should be a pipeline status constant (pipeline.Status*).
func NewItem(taskName, pipelineStatus string, stepsCompleted map[string]pipeline.StepResult, pausedAtStep int) *Item {
	var randBytes [4]byte
	_, _ = rand.Read(randBytes[:])
	randHex := hex.EncodeToString(randBytes[:])

	now := time.Now()

	status := StatusDone
	switch pipelineStatus {
	case pipeline.StatusPaused:
		status = StatusPending
	case pipeline.StatusFailed:
		status = StatusFailed
	}

	return &Item{
		ID:             fmt.Sprintf("%s-%s-%s", now.Format("20060102-150405"), randHex, taskName),
		Task:           taskName,
		Created:        now,
		PausedAtStep:   pausedAtStep,
		StepsCompleted: stepsCompleted,
		Status:         status,
	}
}

// MergeSteps copies step results into this item.
func (item *Item) MergeSteps(steps map[string]pipeline.StepResult) {
	if item.StepsCompleted == nil {
		item.StepsCompleted = make(map[string]pipeline.StepResult, len(steps))
	}
	for k, v := range steps {
		item.StepsCompleted[k] = v
	}
}

// StepNames returns the names of completed steps in pipeline order. Steps not
// in tk (all of them when tk is nil) follow in alphabetical order.
func (item *Item) StepNames(tk *task.Task) []string {
	names := make([]string, 0, len(item.StepsCompleted))
	if tk != nil {
		for _, s := range tk.Steps {
			if _, ok := item.StepsCompleted[s.Name]; ok {
				names = append(names, s.Name)
			}
		}
	}
	var rest []string
	for name := range item.StepsCompleted {
		if !slices.Contains(names, name) {
			rest = append(rest, name)
		}
	}
	slices.Sort(rest)
	return append(names, rest...)
}

// StepLabel describes the step the item stopped at by name and 1-based
// position, e.g. "publish (3 of 3)". It returns "-" for completed items.
// tk may be nil when the task definition is no longer available.
func (item *Item) StepLabel(tk *task.Task) string {
	i := item.PausedAtStep
	if item.Status == StatusDone {
		return "-"
	}
	if tk == nil {
		return fmt.Sprintf("step %d", i+1)
	}
	if i < 0 || i >= len(tk.Steps) {
		return "-"
	}
	return fmt.Sprintf("%s (%d of %d)", tk.Steps[i].Name, i+1, len(tk.Steps))
}

type Queue struct {
	dir string
}

func New(dir string) *Queue {
	return &Queue{dir: dir}
}

func (q *Queue) ensureDir() error {
	return os.MkdirAll(q.dir, 0o755)
}

func (q *Queue) Save(item *Item) error {
	if err := q.ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(q.dir, item.ID+".json")
	return os.WriteFile(path, data, 0o644)
}

func (q *Queue) Load(id string) (*Item, error) {
	path := filepath.Join(q.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (q *Queue) ListPending() ([]*Item, error) {
	return q.List(StatusPending)
}

// List returns items with any of the given statuses, or all items when none
// are given. Stale running items are marked failed first.
func (q *Queue) List(statuses ...string) ([]*Item, error) {
	if err := q.RecoverStale(); err != nil {
		return nil, err
	}

	all, err := q.ListAll()
	if err != nil {
		return nil, err
	}
	if len(statuses) == 0 {
		return all, nil
	}

	var matched []*Item
	for _, item := range all {
		if slices.Contains(statuses, item.Status) {
			matched = append(matched, item)
		}
	}
	return matched, nil
}

// RecoverStale marks running items whose PID is no longer alive as failed.
func (q *Queue) RecoverStale() error {
	all, err := q.ListAll()
	if err != nil {
		return err
	}

	for _, item := range all {
		if item.Status != StatusRunning || item.PID == 0 {
			continue
		}
		proc, err := os.FindProcess(item.PID)
		if err != nil {
			// Process doesn't exist — mark as failed.
			item.Status = StatusFailed
			item.PID = 0
			if err := q.Save(item); err != nil {
				return err
			}
			continue
		}
		// On Unix, FindProcess always succeeds. Send signal 0 to check liveness.
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			item.Status = StatusFailed
			item.PID = 0
			if err := q.Save(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (q *Queue) Delete(id string) error {
	path := filepath.Join(q.dir, id+".json")
	return os.Remove(path)
}

func (q *Queue) DeleteByStatus(statuses ...string) (int, error) {
	all, err := q.ListAll()
	if err != nil {
		return 0, err
	}

	statusSet := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		statusSet[s] = true
	}

	deleted := 0
	for _, item := range all {
		if statusSet[item.Status] {
			if err := q.Delete(item.ID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	return deleted, nil
}

// DeleteFinished removes items that will not run again without user action:
// done, failed, and rejected.
func (q *Queue) DeleteFinished() (int, error) {
	return q.DeleteByStatus(StatusDone, StatusFailed, StatusRejected)
}

func (q *Queue) DeleteByTask(taskName string) (int, error) {
	all, err := q.ListAll()
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, item := range all {
		if item.Task == taskName {
			if err := q.Delete(item.ID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	return deleted, nil
}

func (q *Queue) ListAll() ([]*Item, error) {
	entries, err := os.ReadDir(q.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var items []*Item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		item, err := q.Load(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
