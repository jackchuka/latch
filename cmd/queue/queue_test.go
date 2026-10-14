package queuecmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/pflag"
)

const testTask = `name: digest
steps:
  - name: fetch
    command: "true"
  - name: summarize
    command: "true"
  - name: analyze
    command: "true"
  - name: post
    command: "true"
    approve: true
`

func setup(t *testing.T) *queue.Queue {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p, err := paths.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.TasksDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.TaskFile("digest"), []byte(testTask), 0o644); err != nil {
		t.Fatal(err)
	}
	return queue.New(p.QueueDir())
}

func save(t *testing.T, q *queue.Queue, id, status string, pausedAt int, steps ...string) {
	t.Helper()
	completed := make(map[string]pipeline.StepResult)
	for _, s := range steps {
		completed[s] = pipeline.StepResult{Output: s + " output"}
	}
	item := &queue.Item{ID: id, Task: "digest", Created: time.Now(), Status: status, PausedAtStep: pausedAt, StepsCompleted: completed}
	if err := q.Save(item); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	t.Cleanup(func() {
		for _, c := range Cmd.Commands() {
			c.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			})
		}
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	Cmd.SetArgs(args)
	runErr := Cmd.Execute()
	os.Stdout = stdout
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatalf("queue %v: %v", args, runErr)
	}
	return buf.String()
}

func TestShowStepsInPipelineOrder(t *testing.T) {
	q := setup(t)
	save(t, q, "item1", queue.StatusPending, 3, "fetch", "summarize", "analyze")

	out := run(t, "show", "item1")

	var positions []int
	for _, name := range []string{"fetch", "summarize", "analyze"} {
		positions = append(positions, strings.Index(out, "--- step: "+name+" ---"))
	}
	if positions[0] < 0 || positions[0] > positions[1] || positions[1] > positions[2] {
		t.Errorf("steps not in pipeline order (fetch, summarize, analyze):\n%s", out)
	}
}

func TestShowPausedStepByName(t *testing.T) {
	q := setup(t)
	save(t, q, "item1", queue.StatusPending, 3, "fetch", "summarize", "analyze")
	save(t, q, "item2", queue.StatusFailed, 1, "fetch")

	if out := run(t, "show", "item1"); !strings.Contains(out, "Paused At Step: post (4 of 4)") {
		t.Errorf("pending item should name the paused step, got:\n%s", out)
	}
	if out := run(t, "show", "item2"); !strings.Contains(out, "Failed At Step: summarize (2 of 4)") {
		t.Errorf("failed item should name the failed step, got:\n%s", out)
	}
}

func TestListStatusFilter(t *testing.T) {
	q := setup(t)
	save(t, q, "pending1", queue.StatusPending, 3, "fetch", "summarize", "analyze")
	save(t, q, "failed1", queue.StatusFailed, 1, "fetch")
	save(t, q, "done1", queue.StatusDone, 4, "fetch", "summarize", "analyze", "post")

	out := run(t, "list")
	if !strings.Contains(out, "pending1") || strings.Contains(out, "failed1") {
		t.Errorf("default list should show only pending items, got:\n%s", out)
	}
	if !strings.Contains(out, "post") {
		t.Errorf("list should name the paused step, got:\n%s", out)
	}

	out = run(t, "list", "--status", "failed")
	if !strings.Contains(out, "failed1") || strings.Contains(out, "pending1") || strings.Contains(out, "done1") {
		t.Errorf("--status failed should show only failed items, got:\n%s", out)
	}

	out = run(t, "list", "--status", "failed,done")
	if !strings.Contains(out, "failed1") || !strings.Contains(out, "done1") || strings.Contains(out, "pending1") {
		t.Errorf("--status failed,done should show failed and done items, got:\n%s", out)
	}

	out = run(t, "list", "--status", "all")
	for _, id := range []string{"pending1", "failed1", "done1"} {
		if !strings.Contains(out, id) {
			t.Errorf("--status all should include %s, got:\n%s", id, out)
		}
	}

	Cmd.SetArgs([]string{"list", "--status", "bogus"})
	Cmd.SilenceUsage, Cmd.SilenceErrors = true, true
	if err := Cmd.Execute(); err == nil {
		t.Error("--status bogus should be rejected")
	}
}

func TestClearRemovesFailed(t *testing.T) {
	q := setup(t)
	save(t, q, "pending1", queue.StatusPending, 3)
	save(t, q, "failed1", queue.StatusFailed, 1)
	save(t, q, "rejected1", queue.StatusRejected, 3)

	run(t, "clear")

	all, err := q.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "pending1" {
		var ids []string
		for _, it := range all {
			ids = append(ids, it.ID)
		}
		t.Errorf("after clear, remaining = %v, want [pending1]", ids)
	}

}

func TestListEmptyJSON(t *testing.T) {
	setup(t)
	if out := strings.TrimSpace(run(t, "list", "--status", "failed", "-o", "json")); out != "[]" {
		t.Errorf("empty JSON list = %q, want []", out)
	}
}
