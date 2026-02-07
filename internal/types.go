package internal

import "strings"

// StdinData is the top-level JSON structure piped by Claude Code every ~300ms.
type StdinData struct {
	SessionID      string        `json:"session_id"`
	TranscriptPath string        `json:"transcript_path"`
	HookEventName  string        `json:"hook_event_name"`
	CWD            string        `json:"cwd"`
	Version        string        `json:"version"`
	Exceeds200K    bool          `json:"exceeds_200k_tokens"`
	Model          Model         `json:"model"`
	Workspace      Workspace     `json:"workspace"`
	Cost           Cost          `json:"cost"`
	ContextWindow  ContextWindow `json:"context_window"`
	OutputStyle    *OutputStyle  `json:"output_style"`
	Vim            *Vim          `json:"vim"`
	Agent          *Agent        `json:"agent"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

type Cost struct {
	TotalCostUSD       float64 `json:"total_cost_usd"`
	TotalDurationMS    int64   `json:"total_duration_ms"`
	TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

type ContextWindow struct {
	TotalInputTokens    int           `json:"total_input_tokens"`
	TotalOutputTokens   int           `json:"total_output_tokens"`
	ContextWindowSize   int           `json:"context_window_size"`
	UsedPercentage      *float64      `json:"used_percentage"`
	RemainingPercentage *float64      `json:"remaining_percentage"`
	CurrentUsage        *CurrentUsage `json:"current_usage"`
}

type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type OutputStyle struct {
	Name string `json:"name"`
}

type Vim struct {
	Mode string `json:"mode"`
}

type Agent struct {
	Name string `json:"name"`
}

// ModelTier classifies a model by its performance/cost tier.
type ModelTier int

const (
	TierUnknown ModelTier = iota
	TierHaiku
	TierSonnet
	TierOpus
)

func classifyModel(m Model) ModelTier {
	name := m.DisplayName
	if name == "" {
		name = m.ID
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "opus") {
		return TierOpus
	}
	if strings.Contains(lower, "sonnet") {
		return TierSonnet
	}
	if strings.Contains(lower, "haiku") {
		return TierHaiku
	}
	return TierUnknown
}
