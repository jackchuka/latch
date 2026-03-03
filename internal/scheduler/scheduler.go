package scheduler

// Scheduler abstracts platform-specific task scheduling backends
// (launchd on macOS, systemd on Linux, etc.).
type Scheduler interface {
	Install(taskName, schedule string) error
	Uninstall(taskName string) error
	Installed(taskName string) bool
}
