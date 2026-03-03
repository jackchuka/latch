package launchd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackchuka/latch/internal/tmpl"
)

// CalendarInterval represents a single launchd StartCalendarInterval dict.
// Each field has a bool indicating whether the value is set, and the value itself.
type CalendarInterval struct {
	Minute     bool
	MinuteVal  int
	Hour       bool
	HourVal    int
	Weekday    bool
	WeekdayVal int
	Day        bool
	DayVal     int
	Month      bool
	MonthVal   int
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.latch.{{.TaskName}}</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>{{.Path}}</string>
	</dict>
	<key>ProgramArguments</key>
	<array>
		<string>{{.BinaryPath}}</string>
		<string>task</string>
		<string>run</string>
		<string>{{.TaskName}}</string>
	</array>
	<key>StartCalendarInterval</key>
	<array>
{{- range .Intervals}}
		<dict>
{{- if .Minute}}
			<key>Minute</key>
			<integer>{{.MinuteVal}}</integer>
{{- end}}
{{- if .Hour}}
			<key>Hour</key>
			<integer>{{.HourVal}}</integer>
{{- end}}
{{- if .Weekday}}
			<key>Weekday</key>
			<integer>{{.WeekdayVal}}</integer>
{{- end}}
{{- if .Day}}
			<key>Day</key>
			<integer>{{.DayVal}}</integer>
{{- end}}
{{- if .Month}}
			<key>Month</key>
			<integer>{{.MonthVal}}</integer>
{{- end}}
		</dict>
{{- end}}
	</array>
</dict>
</plist>
`

type plistData struct {
	TaskName   string
	BinaryPath string
	Path       string
	Intervals  []CalendarInterval
}

// cronToCalendarIntervals parses a 5-field cron expression and returns
// a slice of CalendarInterval values suitable for launchd's StartCalendarInterval.
func cronToCalendarIntervals(cron string) ([]CalendarInterval, error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 cron fields, got %d", len(fields))
	}

	minuteField := fields[0]
	hourField := fields[1]
	dayField := fields[2]
	monthField := fields[3]
	weekdayField := fields[4]

	minutes, err := expandCronField(minuteField, 0, 59)
	if err != nil {
		return nil, fmt.Errorf("parsing minute field: %w", err)
	}

	hours, err := expandCronField(hourField, 0, 23)
	if err != nil {
		return nil, fmt.Errorf("parsing hour field: %w", err)
	}

	days, err := expandCronField(dayField, 1, 31)
	if err != nil {
		return nil, fmt.Errorf("parsing day field: %w", err)
	}

	months, err := expandCronField(monthField, 1, 12)
	if err != nil {
		return nil, fmt.Errorf("parsing month field: %w", err)
	}

	weekdays, err := expandCronField(weekdayField, 0, 6)
	if err != nil {
		return nil, fmt.Errorf("parsing weekday field: %w", err)
	}

	// Build the Cartesian product of all expanded field values.
	intervals := []CalendarInterval{{}}

	type fieldSetter struct {
		values []int
		set    func(*CalendarInterval, int)
	}

	fieldsToExpand := []fieldSetter{
		{minutes, func(ci *CalendarInterval, v int) { ci.Minute = true; ci.MinuteVal = v }},
		{hours, func(ci *CalendarInterval, v int) { ci.Hour = true; ci.HourVal = v }},
		{days, func(ci *CalendarInterval, v int) { ci.Day = true; ci.DayVal = v }},
		{months, func(ci *CalendarInterval, v int) { ci.Month = true; ci.MonthVal = v }},
		{weekdays, func(ci *CalendarInterval, v int) { ci.Weekday = true; ci.WeekdayVal = v }},
	}

	for _, fs := range fieldsToExpand {
		if fs.values == nil {
			continue
		}
		var expanded []CalendarInterval
		for _, ci := range intervals {
			for _, v := range fs.values {
				copy := ci
				fs.set(&copy, v)
				expanded = append(expanded, copy)
			}
		}
		intervals = expanded
	}

	return intervals, nil
}

// expandCronField parses a single cron field and returns the expanded list of
// integer values, or nil if the field is "*" (meaning "any").
// All values must be within [min, max].
func expandCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return nil, nil
	}

	var result []int

	parts := strings.Split(field, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err := strconv.Atoi(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q: %w", bounds[0], err)
			}
			hi, err := strconv.Atoi(bounds[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q: %w", bounds[1], err)
			}
			if lo < min || hi > max {
				return nil, fmt.Errorf("value out of range [%d-%d]: %s", min, max, part)
			}
			for i := lo; i <= hi; i++ {
				result = append(result, i)
			}
		} else {
			val, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value %q: %w", part, err)
			}
			if val < min || val > max {
				return nil, fmt.Errorf("value out of range [%d-%d]: %d", min, max, val)
			}
			result = append(result, val)
		}
	}

	return result, nil
}

// GeneratePlist renders a launchd plist XML string for the given task.
func GeneratePlist(taskName, cron string) (string, error) {
	binPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}

	intervals, err := cronToCalendarIntervals(cron)
	if err != nil {
		return "", fmt.Errorf("parsing cron expression: %w", err)
	}

	return tmpl.Resolve("plist", plistTemplate, plistData{
		TaskName:   taskName,
		BinaryPath: binPath,
		Path:       os.Getenv("PATH"),
		Intervals:  intervals,
	})
}

// Manager handles installing and uninstalling launchd plist files.
type Manager struct {
	dir string
}

// NewManager creates a Manager that writes plist files to dir.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

func (m *Manager) plistPath(taskName string) string {
	return filepath.Join(m.dir, fmt.Sprintf("com.latch.%s.plist", taskName))
}

// Install generates the plist for the task, writes it to the manager's directory,
// and attempts to load it via launchctl. launchctl errors are ignored so that
// Install works in test environments without a running launchd.
func (m *Manager) Install(taskName, cron string) error {
	plist, err := GeneratePlist(taskName, cron)
	if err != nil {
		return err
	}

	path := m.plistPath(taskName)
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return fmt.Errorf("writing plist file: %w", err)
	}

	// Best-effort load; ignore errors for non-macOS or test environments.
	_ = exec.Command("launchctl", "load", path).Run()

	return nil
}

// Installed reports whether a plist file exists for the given task.
func (m *Manager) Installed(taskName string) bool {
	_, err := os.Stat(m.plistPath(taskName))
	return err == nil
}

// Uninstall attempts to unload the plist via launchctl, then deletes the file.
// launchctl errors are ignored so that Uninstall works in test environments.
func (m *Manager) Uninstall(taskName string) error {
	path := m.plistPath(taskName)

	// Best-effort unload; ignore errors for non-macOS or test environments.
	_ = exec.Command("launchctl", "unload", path).Run()

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist file: %w", err)
	}

	return nil
}
