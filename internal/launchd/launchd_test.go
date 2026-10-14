package launchd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePlist(t *testing.T) {
	plist, err := GeneratePlist("daily-standup", "0 9 * * 1-5")
	if err != nil {
		t.Fatalf("GeneratePlist returned error: %v", err)
	}

	if !strings.Contains(plist, "com.latch.daily-standup") {
		t.Error("expected plist to contain label com.latch.daily-standup")
	}
	if !strings.Contains(plist, "<key>Hour</key>") {
		t.Error("expected plist to contain <key>Hour</key>")
	}
	if !strings.Contains(plist, "<integer>9</integer>") {
		t.Error("expected plist to contain <integer>9</integer>")
	}
}

func TestInstallAndUninstall(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.runCmd = func(string, ...string) error { return nil }

	if err := m.Install("test-task", "0 9 * * *"); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	plistPath := filepath.Join(dir, "com.latch.test-task.plist")
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		t.Fatalf("expected plist file to exist at %s", plistPath)
	}

	if err := m.Uninstall("test-task"); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}

	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Fatalf("expected plist file to be removed at %s", plistPath)
	}
}

type cmdCall struct {
	name string
	args []string
}

func TestInstallCallsBootstrap(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	var calls []cmdCall
	m.runCmd = func(name string, args ...string) error {
		calls = append(calls, cmdCall{name, args})
		return nil
	}

	if err := m.Install("my-task", "0 9 * * *"); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	domain := fmt.Sprintf("gui/%d", os.Getuid())
	plistPath := filepath.Join(dir, "com.latch.my-task.plist")

	if len(calls) != 2 {
		t.Fatalf("expected 2 launchctl calls, got %d", len(calls))
	}

	// First call: bootout the old service
	if calls[0].name != "launchctl" {
		t.Errorf("expected launchctl, got %s", calls[0].name)
	}
	wantBootout := []string{"bootout", domain + "/com.latch.my-task"}
	if !equalStrings(calls[0].args, wantBootout) {
		t.Errorf("bootout args = %v, want %v", calls[0].args, wantBootout)
	}

	// Second call: bootstrap the new service
	if calls[1].name != "launchctl" {
		t.Errorf("expected launchctl, got %s", calls[1].name)
	}
	wantBootstrap := []string{"bootstrap", domain, plistPath}
	if !equalStrings(calls[1].args, wantBootstrap) {
		t.Errorf("bootstrap args = %v, want %v", calls[1].args, wantBootstrap)
	}
}

func TestUninstallCallsBootout(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	// Create a plist file so Uninstall has something to remove.
	plistPath := filepath.Join(dir, "com.latch.my-task.plist")
	if err := os.WriteFile(plistPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	var calls []cmdCall
	m.runCmd = func(name string, args ...string) error {
		calls = append(calls, cmdCall{name, args})
		return nil
	}

	if err := m.Uninstall("my-task"); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}

	domain := fmt.Sprintf("gui/%d", os.Getuid())

	if len(calls) != 1 {
		t.Fatalf("expected 1 launchctl call, got %d", len(calls))
	}

	wantBootout := []string{"bootout", domain + "/com.latch.my-task"}
	if !equalStrings(calls[0].args, wantBootout) {
		t.Errorf("bootout args = %v, want %v", calls[0].args, wantBootout)
	}

	// Verify plist file was removed
	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Error("expected plist file to be removed")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCronToCalendarInterval(t *testing.T) {
	tests := []struct {
		name    string
		cron    string
		wantLen int
		checkFn func(t *testing.T, intervals []CalendarInterval)
	}{
		{
			name:    "daily at 9",
			cron:    "0 9 * * *",
			wantLen: 1,
			checkFn: func(t *testing.T, intervals []CalendarInterval) {
				ci := intervals[0]
				if !ci.Minute || ci.MinuteVal != 0 {
					t.Errorf("expected Minute=true, MinuteVal=0, got %v, %d", ci.Minute, ci.MinuteVal)
				}
				if !ci.Hour || ci.HourVal != 9 {
					t.Errorf("expected Hour=true, HourVal=9, got %v, %d", ci.Hour, ci.HourVal)
				}
				if ci.Weekday {
					t.Error("expected Weekday=false for daily schedule")
				}
			},
		},
		{
			name:    "weekdays at 9",
			cron:    "0 9 * * 1-5",
			wantLen: 5,
			checkFn: func(t *testing.T, intervals []CalendarInterval) {
				for i, ci := range intervals {
					if !ci.Weekday {
						t.Errorf("interval %d: expected Weekday=true", i)
					}
					expectedDay := i + 1
					if ci.WeekdayVal != expectedDay {
						t.Errorf("interval %d: expected WeekdayVal=%d, got %d", i, expectedDay, ci.WeekdayVal)
					}
				}
			},
		},
		{
			name:    "every half hour",
			cron:    "0,30 * * * *",
			wantLen: 2,
			checkFn: func(t *testing.T, intervals []CalendarInterval) {
				if intervals[0].MinuteVal != 0 {
					t.Errorf("expected first interval MinuteVal=0, got %d", intervals[0].MinuteVal)
				}
				if intervals[1].MinuteVal != 30 {
					t.Errorf("expected second interval MinuteVal=30, got %d", intervals[1].MinuteVal)
				}
			},
		},
		{
			name:    "multi-value minute and hour (Cartesian product)",
			cron:    "0,30 8,12 * * *",
			wantLen: 4,
			checkFn: func(t *testing.T, intervals []CalendarInterval) {
				type pair struct{ minute, hour int }
				want := []pair{{0, 8}, {0, 12}, {30, 8}, {30, 12}}
				for i, ci := range intervals {
					if ci.MinuteVal != want[i].minute || ci.HourVal != want[i].hour {
						t.Errorf("interval %d: expected minute=%d hour=%d, got minute=%d hour=%d",
							i, want[i].minute, want[i].hour, ci.MinuteVal, ci.HourVal)
					}
				}
			},
		},
		{
			name:    "Monday at 8:30",
			cron:    "30 8 * * 1",
			wantLen: 1,
			checkFn: func(t *testing.T, intervals []CalendarInterval) {
				ci := intervals[0]
				if !ci.Minute || ci.MinuteVal != 30 {
					t.Errorf("expected Minute=true, MinuteVal=30, got %v, %d", ci.Minute, ci.MinuteVal)
				}
				if !ci.Hour || ci.HourVal != 8 {
					t.Errorf("expected Hour=true, HourVal=8, got %v, %d", ci.Hour, ci.HourVal)
				}
				if !ci.Weekday || ci.WeekdayVal != 1 {
					t.Errorf("expected Weekday=true, WeekdayVal=1, got %v, %d", ci.Weekday, ci.WeekdayVal)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervals, err := cronToCalendarIntervals(tt.cron)
			if err != nil {
				t.Fatalf("cronToCalendarIntervals(%q) returned error: %v", tt.cron, err)
			}
			if len(intervals) != tt.wantLen {
				t.Fatalf("expected %d intervals, got %d", tt.wantLen, len(intervals))
			}
			if tt.checkFn != nil {
				tt.checkFn(t, intervals)
			}
		})
	}
}

func TestCronToCalendarIntervalErrors(t *testing.T) {
	tests := []struct {
		name    string
		cron    string
		wantErr string
	}{
		{
			name:    "too few fields",
			cron:    "0 9 *",
			wantErr: "expected 5 cron fields",
		},
		{
			name:    "too many fields",
			cron:    "0 9 * * * *",
			wantErr: "expected 5 cron fields",
		},
		{
			name:    "non-numeric value",
			cron:    "abc 9 * * *",
			wantErr: "invalid value",
		},
		{
			name:    "invalid range start",
			cron:    "abc-5 9 * * *",
			wantErr: "invalid range start",
		},
		{
			name:    "invalid range end",
			cron:    "0 9 * * 1-abc",
			wantErr: "invalid range end",
		},
		{
			name:    "minute out of range",
			cron:    "60 9 * * *",
			wantErr: "value out of range",
		},
		{
			name:    "hour out of range",
			cron:    "0 24 * * *",
			wantErr: "value out of range",
		},
		{
			name:    "day out of range",
			cron:    "0 9 32 * *",
			wantErr: "value out of range",
		},
		{
			name:    "month out of range",
			cron:    "0 9 * 13 *",
			wantErr: "value out of range",
		},
		{
			name:    "weekday out of range",
			cron:    "0 9 * * 8",
			wantErr: "value out of range",
		},
		{
			name:    "range exceeds bounds",
			cron:    "0 9 * * 0-8",
			wantErr: "value out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cronToCalendarIntervals(tt.cron)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestCronToCalendarIntervalStandardSyntax(t *testing.T) {
	type slot struct{ minute, hour, day, month, weekday int } // -1 = unset
	toSlots := func(intervals []CalendarInterval) []slot {
		out := make([]slot, len(intervals))
		for i, ci := range intervals {
			s := slot{-1, -1, -1, -1, -1}
			if ci.Minute {
				s.minute = ci.MinuteVal
			}
			if ci.Hour {
				s.hour = ci.HourVal
			}
			if ci.Day {
				s.day = ci.DayVal
			}
			if ci.Month {
				s.month = ci.MonthVal
			}
			if ci.Weekday {
				s.weekday = ci.WeekdayVal
			}
			out[i] = s
		}
		return out
	}

	tests := []struct {
		cron string
		want []slot
	}{
		{"*/15 * * * *", []slot{{0, -1, -1, -1, -1}, {15, -1, -1, -1, -1}, {30, -1, -1, -1, -1}, {45, -1, -1, -1, -1}}},
		{"0 */8 * * *", []slot{{0, 0, -1, -1, -1}, {0, 8, -1, -1, -1}, {0, 16, -1, -1, -1}}},
		{"0 9 1-10/4 * *", []slot{{0, 9, 1, -1, -1}, {0, 9, 5, -1, -1}, {0, 9, 9, -1, -1}}},
		{"0 0 1 */5 *", []slot{{0, 0, 1, 1, -1}, {0, 0, 1, 6, -1}, {0, 0, 1, 11, -1}}},
		{"0 9 * * 7", []slot{{0, 9, -1, -1, 0}}},
		{"0 9 * * 0,7", []slot{{0, 9, -1, -1, 0}}},
		{"0 9 * * 5-7", []slot{{0, 9, -1, -1, 5}, {0, 9, -1, -1, 6}, {0, 9, -1, -1, 0}}},
		{"0 9 * * */3", []slot{{0, 9, -1, -1, 0}, {0, 9, -1, -1, 3}, {0, 9, -1, -1, 6}}},
		{"0 9 * * 1-5/2", []slot{{0, 9, -1, -1, 1}, {0, 9, -1, -1, 3}, {0, 9, -1, -1, 5}}},
		{"*/1 9 * * *", []slot{{-1, 9, -1, -1, -1}}},
		{"0 9 */1 * 1", []slot{{0, 9, -1, -1, 1}}},
		{"0,30 9 * * 1,3-4", []slot{{0, 9, -1, -1, 1}, {0, 9, -1, -1, 3}, {0, 9, -1, -1, 4}, {30, 9, -1, -1, 1}, {30, 9, -1, -1, 3}, {30, 9, -1, -1, 4}}},
	}

	for _, tt := range tests {
		t.Run(tt.cron, func(t *testing.T) {
			intervals, err := cronToCalendarIntervals(tt.cron)
			if err != nil {
				t.Fatalf("cronToCalendarIntervals(%q) returned error: %v", tt.cron, err)
			}
			got := toSlots(intervals)
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("cronToCalendarIntervals(%q)\n got  %v\n want %v", tt.cron, got, tt.want)
			}
		})
	}
}

func TestCronToCalendarIntervalStandardSyntaxErrors(t *testing.T) {
	tests := []struct {
		cron    string
		wantErr string
	}{
		{"*/0 * * * *", "invalid step"},
		{"*/x * * * *", "invalid step"},
		{"0 9 * * 8", "value out of range"},
		{"0 9 * * 5-1", "invalid range"},
		// cron ANDs a starred day field with a restricted one, which launchd cannot express
		{"0 9 */2 * 1", "cannot be combined"},
		{"0 9 1 * */2", "cannot be combined"},
	}
	for _, tt := range tests {
		t.Run(tt.cron, func(t *testing.T) {
			_, err := cronToCalendarIntervals(tt.cron)
			if err == nil {
				t.Fatalf("cronToCalendarIntervals(%q) returned nil error", tt.cron)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
