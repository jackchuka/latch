package paths_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackchuka/latch/internal/paths"
)

func TestPaths(t *testing.T) {
	p, err := paths.New()
	if err != nil {
		t.Fatalf("paths.New() error: %v", err)
	}

	if !strings.HasSuffix(p.ConfigDir(), filepath.Join("latch")) {
		t.Errorf("ConfigDir should end with 'latch', got %q", p.ConfigDir())
	}

	if !strings.HasSuffix(p.TasksDir(), filepath.Join("latch", "tasks")) {
		t.Errorf("TasksDir should end with 'latch/tasks', got %q", p.TasksDir())
	}

	if !strings.HasSuffix(p.QueueDir(), filepath.Join("latch", "queue")) {
		t.Errorf("QueueDir should end with 'latch/queue', got %q", p.QueueDir())
	}

	if !strings.HasSuffix(p.DataDir(), filepath.Join("latch")) {
		t.Errorf("DataDir should end with 'latch', got %q", p.DataDir())
	}
}

func TestPathsWithOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
	t.Setenv("XDG_DATA_HOME", "/tmp/test-data")

	p, err := paths.New()
	if err != nil {
		t.Fatalf("paths.New() error: %v", err)
	}

	if got := p.ConfigDir(); got != "/tmp/test-config/latch" {
		t.Errorf("ConfigDir = %q, want %q", got, "/tmp/test-config/latch")
	}

	if got := p.DataDir(); got != "/tmp/test-data/latch" {
		t.Errorf("DataDir = %q, want %q", got, "/tmp/test-data/latch")
	}
}

func TestTaskFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
	p, err := paths.New()
	if err != nil {
		t.Fatalf("paths.New() error: %v", err)
	}

	got := p.TaskFile("daily-standup")
	want := "/tmp/test-config/latch/tasks/daily-standup.yaml"
	if got != want {
		t.Errorf("TaskFile = %q, want %q", got, want)
	}
}

func TestEnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))

	p, err := paths.New()
	if err != nil {
		t.Fatalf("paths.New() error: %v", err)
	}

	if err := p.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error: %v", err)
	}

	for _, dir := range []string{p.TasksDir(), p.QueueDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %q was not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}
}
