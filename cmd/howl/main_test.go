package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary to temp dir
	tmpDir, err := os.MkdirTemp("", "howl-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "howl")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "." // cmd/howl/ directory (test runs from package dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runBinary(t *testing.T, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(stdin)
	// Isolate HOME to prevent loading real config
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run binary: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func countNonEmptyLines(s string) int {
	lines := strings.Split(s, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func TestE2E_VersionFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		flag string
	}{
		{"long", "--version"},
		{"short", "-v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stdout, stderr, exitCode := runBinary(t, "", tt.flag)

			if exitCode != 0 {
				t.Errorf("exitCode = %d, want 0", exitCode)
			}

			if stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}

			if !strings.HasPrefix(stdout, "howl ") {
				t.Errorf("stdout = %q, want prefix 'howl '", stdout)
			}

			if !strings.Contains(stdout, "(") || !strings.Contains(stdout, ")") {
				t.Errorf("stdout = %q, want format 'howl VERSION (COMMIT)'", stdout)
			}
		})
	}
}

func TestE2E_HelpFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		flag string
	}{
		{"long", "--help"},
		{"short", "-h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stdout, stderr, exitCode := runBinary(t, "", tt.flag)

			if exitCode != 0 {
				t.Errorf("exitCode = %d, want 0", exitCode)
			}

			if stdout != "" {
				t.Errorf("stdout = %q, want empty (help goes to stderr)", stdout)
			}

			if !strings.Contains(stderr, "howl:") {
				t.Errorf("stderr = %q, want help message containing 'howl:'", stderr)
			}

			if !strings.Contains(stderr, "stdin") {
				t.Errorf("stderr = %q, want help message containing 'stdin'", stderr)
			}
		})
	}
}

func TestE2E_InvalidJSON(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runBinary(t, "not json")

	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}

	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "stdin parse error") {
		t.Errorf("stderr = %q, want 'stdin parse error'", stderr)
	}
}

func TestE2E_ValidJSON_MinimalData(t *testing.T) {
	t.Parallel()

	input := `{
		"model": {"display_name": "Sonnet"},
		"context_window": {"context_window_size": 200000},
		"cost": {"total_duration_ms": 60000}
	}`

	stdout, stderr, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	if stdout == "" {
		t.Error("stdout is empty, want non-empty output")
	}
}

func TestE2E_OutputFormat(t *testing.T) {
	t.Parallel()

	input := `{
		"model": {"display_name": "Sonnet"},
		"context_window": {"context_window_size": 200000},
		"cost": {"total_duration_ms": 60000}
	}`

	stdout, _, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}

	lines := strings.Split(stdout, "\n")
	const resetPrefix = "\033[0m"

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // Skip empty lines
		}

		if !strings.HasPrefix(line, resetPrefix) {
			t.Errorf("line %d = %q, want prefix %q", i, line, resetPrefix)
		}
	}
}

func TestE2E_NBSPReplacement(t *testing.T) {
	t.Parallel()

	input := `{
		"model": {"display_name": "Sonnet"},
		"context_window": {"context_window_size": 200000},
		"cost": {"total_duration_ms": 60000}
	}`

	stdout, _, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}

	// Check for NBSP (U+00A0)
	if !strings.Contains(stdout, "\u00A0") {
		t.Error("output does not contain NBSP (\\u00A0)")
	}

	// After the Reset prefix, content should have NBSP, not regular spaces
	// Extract content after first Reset prefix
	const resetPrefix = "\033[0m"
	idx := strings.Index(stdout, resetPrefix)
	if idx == -1 {
		t.Fatal("output does not contain Reset prefix")
	}

	contentAfterReset := stdout[idx+len(resetPrefix):]

	// Count regular spaces between ANSI codes and content
	// The separator " | " becomes "\u00A0|\u00A0" after replacement
	if strings.Contains(contentAfterReset, " | ") {
		t.Error("output contains regular space separator ' | ', expected NBSP separator")
	}
}

func TestE2E_DangerMode(t *testing.T) {
	t.Parallel()

	input := `{
		"model": {"display_name": "Sonnet"},
		"context_window": {
			"context_window_size": 200000,
			"used_percentage": 92
		},
		"cost": {"total_duration_ms": 60000}
	}`

	stdout, _, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}

	lineCount := countNonEmptyLines(stdout)
	if lineCount != 2 {
		t.Errorf("line count = %d, want 2 (danger mode)", lineCount)
	}

	if !strings.Contains(stdout, "🔴") {
		t.Error("output does not contain danger emoji 🔴")
	}
}

func TestE2E_FullSessionData(t *testing.T) {
	t.Parallel()

	input := `{
		"session_id": "test-session",
		"model": {"id": "claude-opus-4-6", "display_name": "Claude Opus 4.6"},
		"context_window": {
			"context_window_size": 200000,
			"used_percentage": 45.0,
			"current_usage": {
				"input_tokens": 50000,
				"output_tokens": 5000,
				"cache_read_input_tokens": 30000
			}
		},
		"cost": {
			"total_cost_usd": 2.50,
			"total_duration_ms": 180000,
			"total_api_duration_ms": 120000,
			"total_lines_added": 100,
			"total_lines_removed": 20
		},
		"workspace": {"current_dir": "/test/project"}
	}`

	stdout, stderr, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	if !strings.Contains(stdout, "Opus") {
		t.Error("output does not contain model name 'Opus'")
	}

	if !strings.Contains(stdout, "$2.50") {
		t.Error("output does not contain cost '$2.50'")
	}
}

func TestE2E_EmptyJSON(t *testing.T) {
	t.Parallel()

	input := `{}`

	stdout, stderr, exitCode := runBinary(t, input)

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	if stdout == "" {
		t.Error("stdout is empty, want some output (no panic)")
	}
}

func TestE2E_SubagentRows(t *testing.T) {
	t.Parallel()

	input := `{
		"session_id": "s",
		"columns": 80,
		"tasks": [
			{"id": "t1", "name": "explore", "status": "running",
			 "contextWindowSize": 200000, "tokenCount": 50000, "effort": "high"},
			{"id": "t2", "name": "executor", "effort": 32000},
			{"name": "no-id", "status": "running"}
		]
	}`

	stdout, stderr, exitCode := runBinary(t, input, "--subagent")

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0 (stderr: %q)", exitCode, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d rows, want 2 (the task without an id is skipped): %q", len(lines), stdout)
	}

	wantIDs := []string{"t1", "t2"}
	for i, line := range lines {
		var row struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("row %d is not valid JSON: %v (%q)", i, err, line)
		}
		if row.ID != wantIDs[i] {
			t.Errorf("row %d id = %q, want %q", i, row.ID, wantIDs[i])
		}
		if row.Content == "" {
			t.Errorf("row %d has empty content", i)
		}
		if strings.Contains(row.Content, " ") {
			t.Errorf("row %d content contains a plain space, want NBSP: %q", i, row.Content)
		}
	}

	// 50000/200000 is the per-row context percentage Claude Code cannot render
	// by default; it is the reason this mode exists.
	if !strings.Contains(lines[0], "25%") {
		t.Errorf("row t1 missing per-row context percentage: %q", lines[0])
	}
}

func TestE2E_SubagentInvalidJSON(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runBinary(t, "not json", "--subagent")

	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "subagent stdin parse error") {
		t.Errorf("stderr = %q, want 'subagent stdin parse error'", stderr)
	}
}
