package internal

import "strings"

// StdinData represents the top-level JSON structure piped by Claude Code every ~300ms.
// It contains session metrics, context window usage, cost data, and workspace information.
// Field set tracks the Claude Code 2.1 statusline schema (https://code.claude.com/docs/en/statusline).
type StdinData struct {
	SessionID         string        `json:"session_id"`
	SessionName       string        `json:"session_name"`
	TranscriptPath    string        `json:"transcript_path"`
	CWD               string        `json:"cwd"`
	Version           string        `json:"version"`
	Model             Model         `json:"model"`
	Workspace         Workspace     `json:"workspace"`
	Cost              Cost          `json:"cost"`
	ContextWindow     ContextWindow `json:"context_window"`
	Exceeds200KTokens bool          `json:"exceeds_200k_tokens"`
	FastMode          bool          `json:"fast_mode"`
	OutputStyle       *OutputStyle  `json:"output_style"`
	Vim               *Vim          `json:"vim"`
	Agent             *Agent        `json:"agent"`
	Effort            *Effort       `json:"effort"`
	Thinking          *Thinking     `json:"thinking"`
	RateLimits        *RateLimits   `json:"rate_limits"`
	PromptCache       *PromptCache  `json:"prompt_cache"`
	PR                *PullRequest  `json:"pr"`
	Worktree          *Worktree     `json:"worktree"`
}

// Model represents the AI model being used for the session.
type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Workspace represents the working directory context for the session.
type Workspace struct {
	CurrentDir  string    `json:"current_dir"`
	ProjectDir  string    `json:"project_dir"`
	AddedDirs   []string  `json:"added_dirs"`
	GitWorktree string    `json:"git_worktree"`
	Repo        *RepoInfo `json:"repo"`
}

// RepoInfo identifies the git repository hosting the workspace, when an origin
// remote is present.
type RepoInfo struct {
	Host  string `json:"host"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
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

// OutputStyle represents the current output style configuration.
type OutputStyle struct {
	Name string `json:"name"`
}

// Vim represents the current vim mode if enabled in Claude Code.
type Vim struct {
	Mode string `json:"mode"`
}

// Agent names the active agent, present when Claude Code runs with the --agent
// flag or with agent settings configured.
type Agent struct {
	Name string `json:"name"`
}

// Effort represents the reasoning effort level, present when the model supports
// the reasoning effort parameter.
type Effort struct {
	Level string `json:"level"` // e.g. "high"
}

// Thinking represents whether extended thinking is enabled.
type Thinking struct {
	Enabled bool `json:"enabled"`
}

// PullRequest represents an open pull request associated with the branch.
type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	ReviewState string `json:"review_state"` // e.g. "pending"; may be absent

	// Kind is "mr" when this describes a GitLab merge request, and absent for
	// GitHub pull requests. On a GitLab remote, Claude Code sets ReviewState to
	// "approved" when the MR is mergeable, "pending" for any other open state,
	// and "draft" for a draft. Requires Claude Code 2.1.234+.
	Kind string `json:"kind"`
}

// Worktree represents an active --worktree session.
type Worktree struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"` // may be absent for hook-based worktrees
	OriginalCWD    string `json:"original_cwd"`
	OriginalBranch string `json:"original_branch"` // may be absent
}

// RateLimits holds Claude.ai subscription rate-limit usage. Present only for
// subscribers, after the first API response. Each window can be independently absent.
type RateLimits struct {
	FiveHour *RateLimitWindow `json:"five_hour"`
	SevenDay *RateLimitWindow `json:"seven_day"`
}

// RateLimitWindow is a single rate-limit window. UsedPercentage is 0-100;
// ResetsAt is Unix epoch seconds.
type RateLimitWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

// PromptCache summarizes how the session's main conversation is using the
// prompt cache. Claude Code derives it from the cache token counts in API
// responses, so it is provider-independent. Absent until the main
// conversation's first API response; subagent requests are not counted.
// Requires Claude Code 2.1.251+.
//
// This is session-cumulative, unlike Metrics.CacheEfficiency, which describes
// only the most recent API call. Neither supersedes the other.
type PromptCache struct {
	Warm            bool   `json:"warm"`
	CachingObserved bool   `json:"caching_observed"`
	TTL             string `json:"ttl"`        // "5m" or "1h"
	ExpiresAt       *int64 `json:"expires_at"` // epoch seconds; nil when the last response reported no cache tokens

	Requests         int `json:"requests"`
	Misses           int `json:"misses"`
	ExpectedRebuilds int `json:"expected_rebuilds"`

	// HitRatio is cache reads over all input tokens this session, 0-1. The
	// denominator counts cache reads, cache writes, and uncached input.
	// nil while those counts are all zero.
	HitRatio *float64 `json:"hit_ratio"`

	CacheWriteTokens  int `json:"cache_write_tokens"`
	MissRecacheTokens int `json:"miss_recache_tokens"`

	LastMissAt    *int64         `json:"last_miss_at"`
	LastMissCause *LastMissCause `json:"last_miss_cause"`
	MissCauses    map[string]int `json:"miss_causes"`

	// RecacheTokensIfCold is what the next request re-caches if the cache goes
	// cold first. nil right after a compaction until the next request.
	RecacheTokensIfCold *int `json:"recache_tokens_if_cold"`
}

// LastMissCause reports what Claude Code identified as the likely cause of the
// most recent cache miss. Requires Claude Code 2.1.260+. nil until the
// session's first miss, and again whenever no cause could be identified.
type LastMissCause struct {
	// Causes names one or more causes, such as "tools_changed",
	// "system_prompt_changed", "ttl_expired_5m", or "likely_server_side".
	Causes []string `json:"causes"`

	ToolsAdded      int `json:"tools_added"`       // with "tools_changed"
	ToolsRemoved    int `json:"tools_removed"`     // with "tools_changed"
	SystemCharDelta int `json:"system_char_delta"` // with "system_prompt_changed"
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
