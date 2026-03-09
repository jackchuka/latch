package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// validName matches identifiers usable in Go text/template: letters, digits, underscores.
var validName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Task struct {
	Name     string `yaml:"name" json:"name"`
	Schedule string `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Steps    []Step `yaml:"steps" json:"steps"`
}

type Step struct {
	Name    string   `yaml:"name" json:"name"`
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args" json:"args,omitempty"`
	Approve bool     `yaml:"approve,omitempty" json:"approve,omitempty"`
}

// StepIndex returns the index of the named step, or -1 if not found.
func (t *Task) StepIndex(name string) int {
	for i, s := range t.Steps {
		if s.Name == name {
			return i
		}
	}
	return -1
}

func (t *Task) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if len(t.Steps) == 0 {
		return fmt.Errorf("task must have at least one step")
	}
	seen := make(map[string]bool)
	for _, s := range t.Steps {
		if s.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if !validName.MatchString(s.Name) {
			return fmt.Errorf("step name %q is not a valid identifier (use letters, digits, underscores)", s.Name)
		}
		if s.Command == "" {
			return fmt.Errorf("step %q: command is required", s.Name)
		}
		if seen[s.Name] {
			return fmt.Errorf("duplicate step name: %q", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}

func Load(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &t, nil
}

func Save(path string, t *Task) error {
	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadAll(dir string) ([]*Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		t, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
