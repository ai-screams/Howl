package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ANSI escape codes for terminal color formatting.
// Reset clears all formatting, bold/dim control intensity,
// colors follow the standard 3-bit palette (30-37),
// and extended 8-bit colors use 38;5;N format.
const (
	Reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	boldRed = "\033[1;31m"
	boldYlw = "\033[1;33m" // gold for opus
	orange  = "\033[38;5;208m"
	grey    = "\033[38;5;245m"
)

// Render produces lines for the statusline display.
// Normal mode: 2-4 lines (depending on active features). Danger mode (configurable, default 85%+): 2 dense lines.
func Render(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo, account *AccountInfo, cfg Config) []string {
	if cfg.Thresholds == (Thresholds{}) {
		cfg.Thresholds = DefaultThresholds()
	}
	if m.ContextPercent >= cfg.Thresholds.ContextDanger {
		return renderDangerMode(d, m, git, usage, tools, cfg.Thresholds)
	}
	return renderNormalMode(d, m, git, usage, tools, account, cfg)
}

func buildLine1(d *StdinData, m Metrics, cfg Config, git *GitInfo, account *AccountInfo) []string {
	t := cfg.Thresholds
	line1 := make([]string, 0, 7)
	line1 = append(line1, renderModelBadge(d.Model, d.ContextWindow.ContextWindowSize))
	if cfg.Features.Account && account != nil && account.EmailAddress != "" {
		line1 = append(line1, renderAccount(account))
	}
	if cfg.Features.Git && git != nil && git.Branch != "" {
		line1 = append(line1, renderGitCompact(git))
	}
	if cfg.Features.ResponseSpeed && m.ResponseSpeed != nil && *m.ResponseSpeed > 0 {
		line1 = append(line1, renderResponseSpeed(*m.ResponseSpeed, t))
	}
	if costStr := renderCost(d.Cost.TotalCostUSD, t); costStr != "" {
		line1 = append(line1, costStr)
	}
	line1 = append(line1, renderDuration(d.Cost.TotalDurationMS))
	return line1
}

func renderNormalMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo, account *AccountInfo, cfg Config) []string {
	t := cfg.Thresholds

	// Line 1: model(+context size) | account | git | speed | cost | duration
	line1 := buildLine1(d, m, cfg, git, account)

	// Line 2: context bar | 5h quota bar | 7d quota bar
	line2 := make([]string, 0, 3)
	line2 = append(line2, renderContextBar(m.ContextPercent, d.ContextWindow, t))
	if cfg.Features.Quota && usage != nil {
		line2 = append(line2, renderQuotaBar(usage.RemainingPercent5h, usage.ResetsAt5h, "5h", t))
		line2 = append(line2, renderQuotaBar(usage.RemainingPercent7d, usage.ResetsAt7d, "7d", t))
	}

	// Line 3: line changes | cache | wait | cost velocity (+ vim/agent conditionally)
	line3 := make([]string, 0, 7)
	if cfg.Features.LineChanges {
		if lines := renderLineChanges(d.Cost); lines != "" {
			line3 = append(line3, lines)
		}
	}
	if cfg.Features.CacheEfficiency && m.CacheEfficiency != nil && *m.CacheEfficiency > 0 {
		line3 = append(line3, renderCacheEfficiencyLabeled(*m.CacheEfficiency, t, d.ContextWindow.CurrentUsage))
	}
	if cfg.Features.APIWaitRatio && m.APIWaitRatio != nil && *m.APIWaitRatio > 0 {
		line3 = append(line3, renderAPIRatioLabeled(*m.APIWaitRatio, t))
	}
	if cfg.Features.CostVelocity && m.CostPerMinute != nil {
		line3 = append(line3, renderCostVelocityLabeled(*m.CostPerMinute, t))
	}
	if cfg.Features.VimMode && d.Vim != nil && d.Vim.Mode != "" {
		line3 = append(line3, renderVimCompact(d.Vim.Mode))
	}
	if cfg.Features.AgentName && d.Agent != nil && d.Agent.Name != "" {
		line3 = append(line3, renderAgentCompact(d.Agent.Name))
	}
	if v := renderVersion(d.Version); v != "" {
		line3 = append(line3, v)
	}

	// Line 4: tools and agents (conditional)
	line4 := make([]string, 0, 3)
	if cfg.Features.Tools && tools != nil && len(tools.Tools) > 0 {
		line4 = append(line4, renderTools(tools.Tools))
	}
	if cfg.Features.Agents && tools != nil && len(tools.Agents) > 0 {
		line4 = append(line4, renderAgents(tools.Agents))
	}

	lines := []string{joinParts(line1)}
	if len(line2) > 0 {
		lines = append(lines, joinParts(line2))
	}
	if len(line3) > 0 {
		lines = append(lines, joinParts(line3))
	}
	if len(line4) > 0 {
		lines = append(lines, joinParts(line4))
	}
	return lines
}

func renderDangerMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, _ *ToolInfo, t Thresholds) []string {
	// Danger mode line 1: model | context bar | cost | duration (original layout)
	line1 := make([]string, 0, 5)
	line1 = append(line1, renderModelBadge(d.Model, d.ContextWindow.ContextWindowSize))
	line1 = append(line1, renderContextBar(m.ContextPercent, d.ContextWindow, t))
	if costStr := renderCost(d.Cost.TotalCostUSD, t); costStr != "" {
		line1 = append(line1, costStr)
	}
	line1 = append(line1, renderDuration(d.Cost.TotalDurationMS))

	// Line 2: workspace/git | changes | token breakdown | speed | metrics
	line2 := make([]string, 0, 10)

	// Workspace + git
	if ws := renderWorkspace(d); ws != "" {
		if git != nil && git.Branch != "" {
			line2 = append(line2, ws+renderGitCompact(git))
		} else {
			line2 = append(line2, ws)
		}
	}

	// Code changes
	if lines := renderLineChanges(d.Cost); lines != "" {
		line2 = append(line2, lines)
	}

	// Token breakdown: In:XXK Out:YYK Cache:ZZK
	if cu := d.ContextWindow.CurrentUsage; cu != nil {
		line2 = append(line2, renderTokenBreakdown(cu))
	}

	// Performance metrics
	if m.ResponseSpeed != nil && *m.ResponseSpeed > 0 {
		line2 = append(line2, renderResponseSpeed(*m.ResponseSpeed, t))
	}
	if m.CacheEfficiency != nil {
		line2 = append(line2, renderCacheEfficiencyCompact(*m.CacheEfficiency, t))
	}
	if m.APIWaitRatio != nil {
		line2 = append(line2, renderAPIRatioCompact(*m.APIWaitRatio, t))
	}
	if m.CostPerMinute != nil {
		// Show hourly cost in danger mode
		hourly := *m.CostPerMinute * 60
		line2 = append(line2, fmt.Sprintf("%s$%.1f/h%s", yellow, hourly, Reset))
	}

	// Vim/Agent
	if d.Vim != nil && d.Vim.Mode != "" {
		line2 = append(line2, renderVimCompact(d.Vim.Mode))
	}
	if d.Agent != nil && d.Agent.Name != "" {
		line2 = append(line2, renderAgentCompact(d.Agent.Name))
	}

	// Quota at the end of line 2 (danger mode)
	if usage != nil {
		line2 = append(line2, renderQuota(usage, t))
	}

	return []string{joinParts(line1), joinParts(line2)}
}

func renderModelBadge(m Model, contextSize int) string {
	name := m.DisplayName
	if name == "" {
		name = m.ID
	}
	if name == "" {
		name = "?"
	}

	// Detect provider
	suffix := ""
	lowerID := strings.ToLower(m.ID)
	if strings.Contains(lowerID, "anthropic.claude-") {
		suffix = " BR" // Bedrock
	} else if strings.Contains(lowerID, "publishers/anthropic") {
		suffix = " VX" // Vertex
	}

	// Append context window size
	if contextSize > 0 {
		suffix += " (" + formatTokenCount(contextSize) + " context)"
	}

	var color string
	switch classifyModel(m) {
	case TierOpus:
		color = boldYlw
	case TierSonnet:
		color = cyan
	case TierHaiku:
		color = green
	default:
		color = dim
	}

	return fmt.Sprintf("%s[%s%s]%s", color, name, suffix, Reset)
}

func renderContextBar(percent int, cw ContextWindow, t Thresholds) string {
	const width = 10
	filled := width * percent / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	var b strings.Builder
	b.Grow(width)
	for i := 0; i < filled; i++ {
		b.WriteRune('█')
	}
	for i := 0; i < empty; i++ {
		b.WriteRune('░')
	}
	bar := b.String()

	color := contextColor(percent, t)
	prefix := ""
	if percent >= t.ContextDanger {
		prefix = "🔴 "
	} else if percent >= t.ContextWarning {
		prefix = "⚠ "
	}

	// Calculate absolute token usage from percentage (more accurate)
	// The percentage is provided by Claude Code and accounts for all token types
	totalTokens := cw.ContextWindowSize
	usedTokens := totalTokens * percent / 100

	return fmt.Sprintf("%s%s%s%s %d%% (%s/%s)", prefix, color, bar, Reset, percent,
		formatTokenCount(usedTokens), formatTokenCount(totalTokens))
}

func contextColor(p int, t Thresholds) string {
	switch {
	case p >= t.ContextDanger:
		return boldRed
	case p >= t.ContextWarning:
		return orange
	case p >= t.ContextModerate:
		return yellow
	default:
		return green
	}
}

func renderCost(usd float64, t Thresholds) string {
	if usd < 0.001 {
		return ""
	}
	var color string
	switch {
	case usd >= t.SessionCostHigh:
		color = boldRed
	case usd >= t.SessionCostMedium:
		color = yellow
	default:
		color = white
	}
	if usd < 10 {
		return fmt.Sprintf("%s$%.2f%s", color, usd, Reset)
	}
	return fmt.Sprintf("%s$%.1f%s", color, usd, Reset)
}

func renderDuration(ms int64) string {
	minutes := ms / 60000
	if minutes < 1 {
		return grey + "<1m" + Reset
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	h := minutes / 60
	rem := minutes % 60
	return fmt.Sprintf("%dh%dm", h, rem)
}

func renderWorkspace(d *StdinData) string {
	dir := d.Workspace.ProjectDir
	if dir == "" {
		dir = d.Workspace.CurrentDir
	}
	if dir == "" {
		return ""
	}
	// Extract last path component
	name := dir
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' {
			name = dir[i+1:]
			break
		}
	}
	if name == "" {
		return ""
	}
	return grey + name + "/" + Reset
}

func renderAccount(account *AccountInfo) string {
	// Display full email in grey/dim color
	return grey + account.EmailAddress + Reset
}

func renderGitCompact(g *GitInfo) string {
	if g.Branch == "" {
		return ""
	}
	dirty := ""
	if g.Dirty {
		dirty = "*"
	}
	return fmt.Sprintf("%s%s%s%s", magenta, g.Branch, dirty, Reset)
}

func renderLineChanges(c Cost) string {
	if c.TotalLinesAdded == 0 && c.TotalLinesRemoved == 0 {
		return ""
	}
	add := formatCount(c.TotalLinesAdded)
	del := formatCount(c.TotalLinesRemoved)
	return fmt.Sprintf("%sΔ%s%s+%s%s/%s-%s%s", grey, Reset, green, add, Reset, red, del, Reset)
}

func cacheColor(pct int, t Thresholds) string {
	switch {
	case pct >= t.CacheExcellent:
		return green
	case pct >= t.CacheGood:
		return yellow
	default:
		return red
	}
}

func renderCacheEfficiencyCompact(pct int, t Thresholds) string {
	return fmt.Sprintf("%sC%d%%%s", cacheColor(pct, t), pct, Reset)
}

func renderCacheEfficiencyLabeled(pct int, t Thresholds, cu *CurrentUsage) string {
	base := fmt.Sprintf("%sCache:%s%d%%%s", grey, cacheColor(pct, t), pct, Reset)
	if cu == nil {
		return base
	}
	w := formatTokenCount(cu.CacheCreationInputTokens)
	r := formatTokenCount(cu.CacheReadInputTokens)
	return fmt.Sprintf("%s%s(W:%s/R:%s)%s", base, grey, w, r, Reset)
}

func apiRatioColor(pct int, t Thresholds) string {
	switch {
	case pct >= t.WaitHigh:
		return red
	case pct >= t.WaitMedium:
		return yellow
	default:
		return green
	}
}

func renderAPIRatioCompact(pct int, t Thresholds) string {
	return fmt.Sprintf("%sA%d%%%s", apiRatioColor(pct, t), pct, Reset)
}

func renderAPIRatioLabeled(pct int, t Thresholds) string {
	return fmt.Sprintf("%sWait:%s%d%%%s", grey, apiRatioColor(pct, t), pct, Reset)
}

func renderCostVelocityLabeled(perMin float64, t Thresholds) string {
	var color string
	switch {
	case perMin >= t.CostVelocityHigh:
		color = boldRed
	case perMin >= t.CostVelocityMedium:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("%sCost:%s$%.2f/m%s", grey, color, perMin, Reset)
}

func renderResponseSpeed(tokPerSec int, t Thresholds) string {
	var color string
	switch {
	case tokPerSec >= t.SpeedFast:
		color = green // fast
	case tokPerSec >= t.SpeedModerate:
		color = yellow // moderate
	default:
		color = orange // slow
	}
	return fmt.Sprintf("%s%dtok/s%s", color, tokPerSec, Reset)
}

func renderVimCompact(mode string) string {
	var color string
	m := strings.ToLower(mode)
	switch m {
	case "normal":
		color = blue
		return color + "Normal" + Reset
	case "insert":
		color = green
		return color + "Insert" + Reset
	case "visual":
		color = magenta
		return color + "Visual" + Reset
	default:
		return ""
	}
}

func renderVersion(version string) string {
	if version == "" {
		return ""
	}
	return grey + "v" + version + Reset
}

func renderAgentCompact(name string) string {
	runes := []rune(name)
	if len(runes) > 8 {
		name = string(runes[:8])
	}
	return cyan + "@" + name + Reset
}

func renderTokenBreakdown(cu *CurrentUsage) string {
	return fmt.Sprintf("%sIn:%s%s %sOut:%s%s %sCache:%s%s",
		grey, formatTokenCount(cu.InputTokens), Reset,
		grey, formatTokenCount(cu.OutputTokens), Reset,
		green, formatTokenCount(cu.CacheReadInputTokens), Reset)
}

func renderTools(tools map[string]int) string {
	if len(tools) == 0 {
		return ""
	}
	// Sort by count and show top 5
	type entry struct {
		name  string
		count int
	}
	entries := make([]entry, 0, len(tools))
	for name, count := range tools {
		entries = append(entries, entry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	if len(entries) > 5 {
		entries = entries[:5]
	}

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%s%s%s(%d)", blue, e.name, Reset, e.count)
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

func renderAgents(agents []string) string {
	if len(agents) == 0 {
		return ""
	}
	// Show first 2 running agents
	display := agents
	if len(display) > 2 {
		display = display[:2]
	}
	result := fmt.Sprintf("%s▶%s", yellow, Reset)
	for i, a := range display {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%s%s%s", cyan, a, Reset)
	}
	return result
}

func formatCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

func formatTokenCount(tokens int) string {
	if tokens >= 1_000_000 {
		m := float64(tokens) / 1_000_000.0
		if m == float64(int(m)) {
			return fmt.Sprintf("%dM", int(m))
		}
		return fmt.Sprintf("%.1fM", m)
	}
	return fmt.Sprintf("%.0fK", float64(tokens)/1000.0)
}

// renderQuota formats 5h/7d remaining percentage for display.
func renderQuota(u *UsageData, t Thresholds) string {
	color5 := quotaColor(u.RemainingPercent5h, t)
	color7 := quotaColor(u.RemainingPercent7d, t)

	now := time.Now()
	until5h := formatTimeUntil(now, u.ResetsAt5h)
	until7d := formatTimeUntil(now, u.ResetsAt7d)

	return fmt.Sprintf("%s(%s)5h:%s %s%.0f%%%s/%s%.0f%%%s %s:7d(%s)%s",
		dim, until5h, Reset,
		color5, u.RemainingPercent5h, Reset,
		color7, u.RemainingPercent7d, Reset,
		grey, until7d, Reset)
}

func renderQuotaBar(remainPct float64, resetTime time.Time, label string, t Thresholds) string {
	const width = 10
	filled := int(remainPct) * width / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var b strings.Builder
	b.Grow(width)
	for i := 0; i < filled; i++ {
		b.WriteRune('█')
	}
	for i := 0; i < empty; i++ {
		b.WriteRune('░')
	}
	bar := b.String()

	color := quotaColor(remainPct, t)
	now := time.Now()
	var until string
	if label == "5h" {
		until = formatTimeUntilWithMinutes(now, resetTime)
	} else {
		until = formatTimeUntil(now, resetTime)
	}

	return fmt.Sprintf("%s%s%s %.0f%% (%s/%s)", color, bar, Reset, remainPct, until, label)
}

func formatTimeUntil(now, target time.Time) string {
	return formatTimeUntilWith(now, target, false)
}

func formatTimeUntilWithMinutes(now, target time.Time) string {
	return formatTimeUntilWith(now, target, true)
}

func formatTimeUntilWith(now, target time.Time, includeMinutes bool) string {
	if target.IsZero() {
		return "?"
	}
	diff := target.Sub(now)
	if diff < 0 {
		return "0"
	}

	hours := int(diff.Hours())
	minutes := int(diff.Minutes()) % 60

	if hours < 24 {
		if includeMinutes && minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	remainHours := hours % 24
	if remainHours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, remainHours)
}

func quotaColor(remaining float64, t Thresholds) string {
	switch {
	case remaining < t.QuotaCritical:
		return boldRed
	case remaining < t.QuotaLow:
		return red
	case remaining < t.QuotaMedium:
		return orange
	case remaining < t.QuotaHigh:
		return yellow
	default:
		return green
	}
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " " + grey + "|" + Reset + " " + parts[i]
	}
	return result
}
