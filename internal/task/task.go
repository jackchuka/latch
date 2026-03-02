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

func Load(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var t Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	for _, s := range t.Steps {
		if !validName.MatchString(s.Name) {
			return nil, fmt.Errorf("step name %q is not a valid identifier (use letters, digits, underscores); referenced in templates as {{.%s.output}}", s.Name, s.Name)
		}
	}
	return &t, nil
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
