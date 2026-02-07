package internal

import (
	"strings"
	"testing"
	"time"
)

func TestSanitizeSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal id", "abc-123-def", "abc-123-def"},
		{"path traversal", "../../../etc/passwd", "passwd"},
		{"absolute path", "/absolute/path/id", "id"},
		{"dot", ".", "."},
		{"double dot", "..", ".."},
		{"empty", "", "."},
		{"nested traversal", "foo/../bar", "bar"},
		{"just slash", "/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeSessionID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSessionID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuotaColor(t *testing.T) {
	t.Parallel()

	// ANSI color constants from constants.go
	const (
		boldRed = "\033[1;31m"
		red     = "\033[31m"
		orange  = "\033[38;5;208m"
		yellow  = "\033[33m"
		green   = "\033[32m"
	)

	tests := []struct {
		name      string
		remaining float64
		want      string
	}{
		{"critical below 10", 5.0, boldRed},
		{"critical at boundary", 9.99, boldRed},
		{"low at 10", 10.0, red},
		{"low below 25", 24.9, red},
		{"medium at 25", 25.0, orange},
		{"medium below 50", 49.9, orange},
		{"high at 50", 50.0, yellow},
		{"high below 75", 74.9, yellow},
		{"healthy at 75", 75.0, green},
		{"healthy at 100", 100.0, green},
		{"zero critical", 0.0, boldRed},
		{"edge case 10.01", 10.01, red},
		{"edge case 25.01", 25.01, orange},
		{"edge case 50.01", 50.01, yellow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := quotaColor(tt.remaining)
			if got != tt.want {
				t.Errorf("quotaColor(%v) = %q, want %q", tt.remaining, got, tt.want)
			}
		})
	}
}

func TestFormatTimeUntil(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		now    time.Time
		target time.Time
		want   string
	}{
		{
			name:   "zero target",
			now:    baseTime,
			target: time.Time{},
			want:   "?",
		},
		{
			name:   "expired past",
			now:    baseTime,
			target: baseTime.Add(-1 * time.Hour),
			want:   "0",
		},
		{
			name:   "expired exact",
			now:    baseTime,
			target: baseTime,
			want:   "0h",
		},
		{
			name:   "3 hours future",
			now:    baseTime,
			target: baseTime.Add(3 * time.Hour),
			want:   "3h",
		},
		{
			name:   "25 hours future",
			now:    baseTime,
			target: baseTime.Add(25 * time.Hour),
			want:   "1d1h",
		},
		{
			name:   "48 hours future",
			now:    baseTime,
			target: baseTime.Add(48 * time.Hour),
			want:   "2d",
		},
		{
			name:   "1 minute future",
			now:    baseTime,
			target: baseTime.Add(1 * time.Minute),
			want:   "0h",
		},
		{
			name:   "23 hours 59 minutes",
			now:    baseTime,
			target: baseTime.Add(23*time.Hour + 59*time.Minute),
			want:   "23h",
		},
		{
			name:   "7 days exact",
			now:    baseTime,
			target: baseTime.Add(7 * 24 * time.Hour),
			want:   "7d",
		},
		{
			name:   "3 days 12 hours",
			now:    baseTime,
			target: baseTime.Add(3*24*time.Hour + 12*time.Hour),
			want:   "3d12h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatTimeUntil(tt.now, tt.target)
			if got != tt.want {
				t.Errorf("formatTimeUntil() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderQuota(t *testing.T) {
	t.Parallel()

	baseTime := time.Now()

	tests := []struct {
		name           string
		usage          *UsageData
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "normal healthy data",
			usage: &UsageData{
				RemainingPercent5h: 80.0,
				RemainingPercent7d: 90.0,
				ResetsAt5h:         baseTime.Add(3 * time.Hour),
				ResetsAt7d:         baseTime.Add(5 * 24 * time.Hour),
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"5h:", ":7d", "80%", "90%"},
		},
		{
			name: "critical 5h",
			usage: &UsageData{
				RemainingPercent5h: 5.0,
				RemainingPercent7d: 50.0,
				ResetsAt5h:         baseTime.Add(1 * time.Hour),
				ResetsAt7d:         baseTime.Add(2 * 24 * time.Hour),
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"\033[1;31m", "5h:", ":7d"},
		},
		{
			name: "low 5h orange 7d",
			usage: &UsageData{
				RemainingPercent5h: 15.0,
				RemainingPercent7d: 30.0,
				ResetsAt5h:         baseTime.Add(2 * time.Hour),
				ResetsAt7d:         baseTime.Add(3 * 24 * time.Hour),
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"\033[31m", "\033[38;5;208m", "5h:", ":7d"},
		},
		{
			name: "medium both",
			usage: &UsageData{
				RemainingPercent5h: 40.0,
				RemainingPercent7d: 45.0,
				ResetsAt5h:         baseTime.Add(4 * time.Hour),
				ResetsAt7d:         baseTime.Add(6 * 24 * time.Hour),
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"\033[38;5;208m", "5h:", ":7d"},
		},
		{
			name: "zero times",
			usage: &UsageData{
				RemainingPercent5h: 50.0,
				RemainingPercent7d: 75.0,
				ResetsAt5h:         time.Time{},
				ResetsAt7d:         time.Time{},
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"5h:", ":7d", "?"},
		},
		{
			name: "expired reset times",
			usage: &UsageData{
				RemainingPercent5h: 10.0,
				RemainingPercent7d: 20.0,
				ResetsAt5h:         baseTime.Add(-1 * time.Hour),
				ResetsAt7d:         baseTime.Add(-1 * time.Hour),
				FetchedAt:          baseTime.Unix(),
			},
			wantContains: []string{"5h:", ":7d", "0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderQuota(tt.usage)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("renderQuota() missing %q in output:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("renderQuota() should not contain %q in output:\n%s", notWant, got)
				}
			}

			// Basic structure validation
			if !strings.Contains(got, "5h:") || !strings.Contains(got, ":7d") {
				t.Errorf("renderQuota() missing expected format markers (5h: or :7d):\n%s", got)
			}
		})
	}
}
