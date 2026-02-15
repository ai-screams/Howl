package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
)

// TranscriptEntry represents a single line in the Claude Code transcript JSONL file.
type TranscriptEntry struct {
	Message struct {
		Content []ContentBlock `json:"content"`
	} `json:"message"`
}

// ContentBlock represents a single content block within a transcript message.
type ContentBlock struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Input     map[string]interface{} `json:"input"`
	ToolUseID string                 `json:"tool_use_id"`
	IsError   bool                   `json:"is_error"`
}

// ToolInfo represents the aggregated tool usage and running agents from the transcript.
type ToolInfo struct {
	Tools  map[string]int // tool name -> count
	Agents []string       // running agent names
}

// ParseTranscript reads the last N lines of transcript to extract recent tools and agents.
// Returns nil on any error (transcript parsing is optional).
// shortenToolName extracts a readable short name from MCP tool names.
// e.g. "mcp__plugin_serena_serena__find_symbol" → "find_symbol"
// Non-MCP tools (Edit, Read, Bash) are returned as-is.
func shortenToolName(name string) string {
	// Split by "__" double-underscore separator
	parts := strings.Split(name, "__")
	if len(parts) < 3 {
		return name
	}

	// Strip plugin prefix entirely, keep only tool name
	return parts[len(parts)-1]
}

func ParseTranscript(path string) *ToolInfo {
	if path == "" {
		return nil
	}

	lines, err := tailLines(path, 64*1024, 100)
	if err != nil {
		return nil
	}

	toolCounts := make(map[string]int)
	runningAgents := make(map[string]bool)
	agentNames := make(map[string]string) // tool_use_id -> agent description

	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry TranscriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		for _, block := range entry.Message.Content {
			if block.Type == "tool_use" && block.Name != "" {
				if block.Name == "Task" {
					// Extract agent info
					subagentType, _ := block.Input["subagent_type"].(string)
					desc, _ := block.Input["description"].(string)
					if subagentType != "" {
						runningAgents[block.ID] = true
						if desc != "" && len(desc) < 30 {
							agentNames[block.ID] = desc
						} else {
							agentNames[block.ID] = subagentType
						}
					}
				} else if block.Name != "TodoWrite" {
					// Count regular tools (skip TodoWrite)
					toolCounts[shortenToolName(block.Name)]++
				}
			} else if block.Type == "tool_result" && block.ToolUseID != "" {
				// Agent completed
				delete(runningAgents, block.ToolUseID)
			}
		}
	}

	// Get top 5 tools
	type toolEntry struct {
		name  string
		count int
	}
	tools := make([]toolEntry, 0, len(toolCounts))
	for name, count := range toolCounts {
		tools = append(tools, toolEntry{name, count})
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].count > tools[j].count
	})
	if len(tools) > 5 {
		tools = tools[:5]
	}

	topTools := make(map[string]int)
	for _, t := range tools {
		topTools[t.name] = t.count
	}

	// Get running agent names
	agents := make([]string, 0, len(runningAgents))
	for id := range runningAgents {
		if name, ok := agentNames[id]; ok {
			agents = append(agents, name)
		}
	}

	return &ToolInfo{
		Tools:  topTools,
		Agents: agents,
	}
}

// tailLines reads at most maxBytes from the end of a file and returns the last maxLines lines.
func tailLines(path string, maxBytes int64, maxLines int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() <= 0 {
		return nil, nil
	}

	start := stat.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	buf := make([]byte, stat.Size()-start)
	n, err := file.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]

	// If we started mid-line, drop the partial prefix.
	if start > 0 {
		if idx := bytes.IndexByte(buf, '\n'); idx >= 0 {
			buf = buf[idx+1:]
		}
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}
