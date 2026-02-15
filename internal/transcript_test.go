package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture returns the path to a testdata fixture file.
// Go test runner sets CWD to the package directory,
// so "testdata/foo.jsonl" resolves correctly.
func fixture(name string) string {
	return filepath.Join("testdata", name)
}

// writeTempTranscript creates a temporary JSONL file for tests
// that need dynamically generated content (e.g., 200+ lines).
func writeTempTranscript(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("failed to write temp transcript: %v", err)
	}
	return path
}

// toolUseLine generates a JSONL line for a tool_use entry.
// Kept for the dynamic 200+ lines edge case test only.
func toolUseLine(name, id string) string {
	entry := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "tool_use", "name": name, "id": id},
			},
		},
	}
	b, _ := json.Marshal(entry)
	return string(b)
}

func TestParseTranscript(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		wantTools  map[string]int
		wantAgents []string
	}{
		{
			name:       "empty file",
			fixture:    "transcript_empty.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{},
		},
		{
			name:       "single tool use",
			fixture:    "transcript_single_tool.jsonl",
			wantTools:  map[string]int{"Read": 1},
			wantAgents: []string{},
		},
		{
			name:       "multiple same tool",
			fixture:    "transcript_multi_same_tool.jsonl",
			wantTools:  map[string]int{"Read": 3},
			wantAgents: []string{},
		},
		{
			name:       "multiple different tools",
			fixture:    "transcript_multi_different.jsonl",
			wantTools:  map[string]int{"Read": 2, "Write": 1, "Edit": 1},
			wantAgents: []string{},
		},
		{
			name:       "task tool tracked as agent",
			fixture:    "transcript_task_agent.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{"Write tests"},
		},
		{
			name:       "task tool result removes agent",
			fixture:    "transcript_task_completed.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{},
		},
		{
			name:       "multiple tasks different states",
			fixture:    "transcript_multi_tasks.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{"Fix bug"},
		},
		{
			name:       "TodoWrite skipped",
			fixture:    "transcript_todowrite_skip.jsonl",
			wantTools:  map[string]int{"Read": 1},
			wantAgents: []string{},
		},
		{
			name:       "malformed JSON skipped",
			fixture:    "transcript_malformed.jsonl",
			wantTools:  map[string]int{"Read": 1, "Write": 1},
			wantAgents: []string{},
		},
		{
			name:       "mixed valid and invalid",
			fixture:    "transcript_mixed.jsonl",
			wantTools:  map[string]int{"Read": 1, "Write": 1, "Edit": 1},
			wantAgents: []string{},
		},
		{
			name:       "more than 5 tools returns top 5",
			fixture:    "transcript_top5.jsonl",
			wantTools:  map[string]int{"Read": 5, "Write": 3},
			wantAgents: []string{},
		},
		{
			name:       "combined tools and agents",
			fixture:    "transcript_combined.jsonl",
			wantTools:  map[string]int{"Read": 1, "Write": 1, "Edit": 1},
			wantAgents: []string{"Write code", "Run tests"},
		},
		{
			name:       "short description under 30 chars",
			fixture:    "transcript_short_desc.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{"Do work"},
		},
		{
			name:       "long description falls back to subagent_type",
			fixture:    "transcript_long_desc.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{"code-writer"},
		},
		{
			name:       "empty content array",
			fixture:    "transcript_empty_content.jsonl",
			wantTools:  map[string]int{},
			wantAgents: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(tt.fixture)
			got := ParseTranscript(path)

			if got == nil {
				t.Fatal("ParseTranscript() returned nil, want non-nil ToolInfo")
			}

			// Check expected tools are present with correct counts
			for tool, wantCount := range tt.wantTools {
				if gotCount, ok := got.Tools[tool]; !ok {
					t.Errorf("tool %q missing from results", tool)
				} else if gotCount != wantCount {
					t.Errorf("tool %q count = %d, want %d", tool, gotCount, wantCount)
				}
			}

			// For top-5 test, verify exactly 5 tools returned
			if tt.name == "more than 5 tools returns top 5" {
				if len(got.Tools) != 5 {
					t.Errorf("got %d tools, want exactly 5\ngot: %v", len(got.Tools), got.Tools)
				}
			} else {
				// Verify no unexpected tools
				if len(got.Tools) != len(tt.wantTools) {
					t.Errorf("got %d tools, want %d\ngot: %v\nwant: %v",
						len(got.Tools), len(tt.wantTools), got.Tools, tt.wantTools)
				}
				for tool := range got.Tools {
					if _, ok := tt.wantTools[tool]; !ok {
						t.Errorf("unexpected tool %q in results", tool)
					}
				}
			}

			// Check agents
			if len(got.Agents) != len(tt.wantAgents) {
				t.Errorf("got %d agents, want %d\ngot: %v\nwant: %v",
					len(got.Agents), len(tt.wantAgents), got.Agents, tt.wantAgents)
			}
			agentSet := make(map[string]bool)
			for _, a := range got.Agents {
				agentSet[a] = true
			}
			for _, want := range tt.wantAgents {
				if !agentSet[want] {
					t.Errorf("agent %q missing from results", want)
				}
			}
		})
	}
}

func TestParseTranscriptEdgeCases(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		if got := ParseTranscript(""); got != nil {
			t.Errorf("ParseTranscript(\"\") = %v, want nil", got)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		if got := ParseTranscript("/nonexistent/path/file.jsonl"); got != nil {
			t.Errorf("ParseTranscript(non-existent) = %v, want nil", got)
		}
	})

	t.Run("over 200 lines keeps last ~100", func(t *testing.T) {
		// This test needs dynamic generation — too large for a static fixture
		lines := make([]string, 250)
		for i := 0; i < 150; i++ {
			lines[i] = toolUseLine("Read", "tool_"+string(rune(i)))
		}
		for i := 150; i < 250; i++ {
			lines[i] = toolUseLine("Write", "tool_"+string(rune(i)))
		}

		path := writeTempTranscript(t, lines)
		got := ParseTranscript(path)
		if got == nil {
			t.Fatal("ParseTranscript() returned nil")
		}

		writeCount := got.Tools["Write"]
		readCount := got.Tools["Read"]
		if writeCount == 0 {
			t.Error("expected Write tools to be counted")
		}
		if writeCount < readCount {
			t.Errorf("expected Write count (%d) >= Read count (%d) when keeping last ~100 lines",
				writeCount, readCount)
		}
	})

	t.Run("tool result for non-existent task", func(t *testing.T) {
		got := ParseTranscript(fixture("transcript_orphan_result.jsonl"))
		if got == nil {
			t.Fatal("ParseTranscript() returned nil")
		}
		if got.Tools["Read"] != 1 {
			t.Errorf("Read count = %d, want 1", got.Tools["Read"])
		}
	})

	t.Run("multiple tool_use in single message", func(t *testing.T) {
		got := ParseTranscript(fixture("transcript_multi_in_message.jsonl"))
		if got == nil {
			t.Fatal("ParseTranscript() returned nil")
		}
		expected := map[string]int{"Read": 1, "Write": 1, "Edit": 1}
		for tool, count := range expected {
			if got.Tools[tool] != count {
				t.Errorf("%s count = %d, want %d", tool, got.Tools[tool], count)
			}
		}
	})
}

func TestShortenToolName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"MCP serena tool", "mcp__plugin_serena_serena__find_symbol", "find_symbol"},
		{"MCP context7 tool", "mcp__plugin_context7_context7__resolve-library-id", "resolve-library-id"},
		{"MCP sequential thinking", "mcp__sequential-thinking__sequentialthinking", "sequentialthinking"},
		{"built-in Edit", "Edit", "Edit"},
		{"built-in Read", "Read", "Read"},
		{"built-in Bash", "Bash", "Bash"},
		{"empty string", "", ""},
		{"single underscore", "some_tool", "some_tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shortenToolName(tt.input)
			if got != tt.want {
				t.Errorf("shortenToolName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
