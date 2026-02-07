package internal

import (
	"bufio"
	"encoding/json"
	"os"
)

type TranscriptEntry struct {
	Message struct {
		Content []ContentBlock `json:"content"`
	} `json:"message"`
}

type ContentBlock struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Input     map[string]interface{} `json:"input"`
	ToolUseID string                 `json:"tool_use_id"`
	IsError   bool                   `json:"is_error"`
}

type ToolInfo struct {
	Tools  map[string]int // tool name -> count
	Agents []string       // running agent names
}

// parseTranscript reads the last N lines of transcript to extract recent tools and agents.
// Returns nil on any error (transcript parsing is optional).
func ParseTranscript(path string) *ToolInfo {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	// Read last 100 lines (reverse would be better but simple forward scan is OK)
	lines := make([]string, 0, 100)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 200 {
			// Keep only last 100
			lines = lines[len(lines)-100:]
		}
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
					toolCounts[block.Name]++
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
	// Simple bubble sort (good enough for small N)
	for i := 0; i < len(tools); i++ {
		for j := i + 1; j < len(tools); j++ {
			if tools[j].count > tools[i].count {
				tools[i], tools[j] = tools[j], tools[i]
			}
		}
	}
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
