//go:build !darwin

package scheduler

import "fmt"

// New returns an error on unsupported platforms.
func New() (Scheduler, error) {
	return nil, fmt.Errorf("scheduling is not supported on this platform; currently only macOS (launchd) is supported")
}
