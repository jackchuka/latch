package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths provides XDG-compliant directory paths for the latch CLI.
type Paths struct {
	configHome string
	dataHome   string
}

// New creates a Paths instance using XDG_CONFIG_HOME and XDG_DATA_HOME
// environment variables, falling back to ~/.config and ~/.local/share
// respectively when unset.
func New() (*Paths, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	return &Paths{
		configHome: configHome,
		dataHome:   dataHome,
	}, nil
}

// ConfigDir returns the latch configuration directory.
func (p *Paths) ConfigDir() string {
	return filepath.Join(p.configHome, "latch")
}

// TasksDir returns the directory where task definitions are stored.
func (p *Paths) TasksDir() string {
	return filepath.Join(p.ConfigDir(), "tasks")
}

// DataDir returns the latch data directory.
func (p *Paths) DataDir() string {
	return filepath.Join(p.dataHome, "latch")
}

// QueueDir returns the directory where queued items are stored.
func (p *Paths) QueueDir() string {
	return filepath.Join(p.DataDir(), "queue")
}

// TaskFile returns the path to a task definition file by name.
func (p *Paths) TaskFile(name string) string {
	return filepath.Join(p.TasksDir(), name+".yaml")
}

// EnsureDirs creates TasksDir and QueueDir if they do not exist.
func (p *Paths) EnsureDirs() error {
	for _, dir := range []string{p.TasksDir(), p.QueueDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
