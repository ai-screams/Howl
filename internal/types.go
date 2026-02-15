package internal

import "strings"

// StdinData represents the top-level JSON structure piped by Claude Code every ~300ms.
// It contains session metrics, context window usage, cost data, and workspace information.
type StdinData struct {
	SessionID      string        `json:"session_id"`
	TranscriptPath string        `json:"transcript_path"`
	CWD            string        `json:"cwd"`
	Version        string        `json:"version"`
	Model          Model         `json:"model"`
	Workspace      Workspace     `json:"workspace"`
	Cost           Cost          `json:"cost"`
	ContextWindow  ContextWindow `json:"context_window"`
	Vim            *Vim          `json:"vim"`
	Agent          *Agent        `json:"agent"`
}

// Model represents the AI model being used for the session.
type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Workspace represents the working directory context for the session.
type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

// Cost represents the cumulative session cost and duration metrics.
type Cost struct {
	TotalCostUSD       float64 `json:"total_cost_usd"`
	TotalDurationMS    int64   `json:"total_duration_ms"`
	TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

// ContextWindow represents the context window usage and limits for the current session.
type ContextWindow struct {
	TotalInputTokens    int           `json:"total_input_tokens"`
	TotalOutputTokens   int           `json:"total_output_tokens"`
	ContextWindowSize   int           `json:"context_window_size"`
	UsedPercentage      *float64      `json:"used_percentage"`
	RemainingPercentage *float64      `json:"remaining_percentage"`
	CurrentUsage        *CurrentUsage `json:"current_usage"`
}

// CurrentUsage represents the token breakdown for the current API call.
type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// Vim represents the current vim mode if enabled in Claude Code.
type Vim struct {
	Mode string `json:"mode"`
}

// Agent represents the active agent teammate if in team mode.
type Agent struct {
	Name string `json:"name"`
}

// RenderContext bundles all inputs for Render. Optional sources (Git, Usage,
// Tools, Account) are nil-safe — Render skips them when nil.
type RenderContext struct {
	Data    *StdinData
	Metrics Metrics
	Git     *GitInfo
	Usage   *UsageData
	Tools   *ToolInfo
	Account *AccountInfo
	Config  Config
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
