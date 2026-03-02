//go:build darwin

package scheduler

import (
	"os"
	"path/filepath"

	"github.com/jackchuka/latch/internal/launchd"
)

var _ Scheduler = (*launchd.Manager)(nil)

// New returns the platform scheduler. On macOS this is a launchd manager
// writing to ~/Library/LaunchAgents.
func New() (Scheduler, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	return launchd.NewManager(dir), nil
}
