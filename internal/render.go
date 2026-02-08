package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ANSI escape codes
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

// render produces lines for the statusline.
// Normal mode: 2-4 lines (depending on active features). Danger mode (85%+): 2 dense lines.
func Render(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo, account *AccountInfo) []string {
	if m.ContextPercent >= DangerThreshold {
		return renderDangerMode(d, m, git, usage, tools)
	}
	return renderNormalMode(d, m, git, usage, tools, account)
}

func buildLine1(d *StdinData, m Metrics) []string {
	line1 := make([]string, 0, 5)
	line1 = append(line1, renderModelBadge(d.Model))
	line1 = append(line1, renderContextBar(m.ContextPercent, d.ContextWindow))
	if costStr := renderCost(d.Cost.TotalCostUSD); costStr != "" {
		line1 = append(line1, costStr)
	}
	line1 = append(line1, renderDuration(d.Cost.TotalDurationMS))
	return line1
}

func renderNormalMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo, account *AccountInfo) []string {
	line1 := buildLine1(d, m)

	// Line 2: account | git | code changes | speed | quota (right side)
	line2 := make([]string, 0, 7)
	if account != nil && account.EmailAddress != "" {
		line2 = append(line2, renderAccount(account))
	}
	if git != nil && git.Branch != "" {
		line2 = append(line2, renderGitCompact(git))
	}
	if lines := renderLineChanges(d.Cost); lines != "" {
		line2 = append(line2, lines)
	}
	if m.ResponseSpeed != nil && *m.ResponseSpeed > 0 {
		line2 = append(line2, renderResponseSpeed(*m.ResponseSpeed))
	}
	if usage != nil {
		line2 = append(line2, renderQuota(usage))
	}

	// Line 3: tools and agents
	line3 := make([]string, 0, 3)
	if tools != nil && len(tools.Tools) > 0 {
		line3 = append(line3, renderTools(tools.Tools))
	}
	if tools != nil && len(tools.Agents) > 0 {
		line3 = append(line3, renderAgents(tools.Agents))
	}

	// Line 4: metrics + vim/agent
	line4 := make([]string, 0, 6)
	if m.CacheEfficiency != nil && *m.CacheEfficiency > 0 {
		line4 = append(line4, renderCacheEfficiencyLabeled(*m.CacheEfficiency))
	}
	if m.APIWaitRatio != nil && *m.APIWaitRatio > 0 {
		line4 = append(line4, renderAPIRatioLabeled(*m.APIWaitRatio))
	}
	if m.CostPerMinute != nil {
		line4 = append(line4, renderCostVelocityLabeled(*m.CostPerMinute))
	}
	if d.Vim != nil && d.Vim.Mode != "" {
		line4 = append(line4, renderVimCompact(d.Vim.Mode))
	}
	if d.Agent != nil && d.Agent.Name != "" {
		line4 = append(line4, renderAgentCompact(d.Agent.Name))
	}

	lines := []string{joinParts(line1), joinParts(line2)}
	if len(line3) > 0 {
		lines = append(lines, joinParts(line3))
	}
	if len(line4) > 0 {
		lines = append(lines, joinParts(line4))
	}
	return lines
}

func renderDangerMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, _ *ToolInfo) []string {
	line1 := buildLine1(d, m)

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
		line2 = append(line2, renderResponseSpeed(*m.ResponseSpeed))
	}
	if m.CacheEfficiency != nil {
		line2 = append(line2, renderCacheEfficiencyCompact(*m.CacheEfficiency))
	}
	if m.APIWaitRatio != nil {
		line2 = append(line2, renderAPIRatioCompact(*m.APIWaitRatio))
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
		line2 = append(line2, renderQuota(usage))
	}

	return []string{joinParts(line1), joinParts(line2)}
}

func renderModelBadge(m Model) string {
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

func renderContextBar(percent int, cw ContextWindow) string {
	const width = 20
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

	color := contextColor(percent)
	prefix := ""
	if percent >= DangerThreshold {
		prefix = "🔴 "
	} else if percent >= 70 {
		prefix = "⚠ "
	}

	// Calculate absolute token usage from percentage (more accurate)
	// The percentage is provided by Claude Code and accounts for all token types
	totalTokens := cw.ContextWindowSize
	usedTokens := totalTokens * percent / 100

	return fmt.Sprintf("%s%s%s%s %d%% (%s/%s)", prefix, color, bar, Reset, percent,
		formatTokenCount(usedTokens), formatTokenCount(totalTokens))
}

func contextColor(p int) string {
	switch {
	case p >= 85:
		return boldRed
	case p >= 70:
		return orange
	case p >= 50:
		return yellow
	default:
		return green
	}
}

func renderCost(usd float64) string {
	if usd < 0.001 {
		return ""
	}
	var color string
	switch {
	case usd >= SessionCostHigh:
		color = boldRed
	case usd >= SessionCostMedium:
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
	return fmt.Sprintf("%s+%s%s/%s-%s%s", green, add, Reset, red, del, Reset)
}

func cacheColor(pct int) string {
	switch {
	case pct >= CacheExcellent:
		return green
	case pct >= CacheGood:
		return yellow
	default:
		return red
	}
}

func renderCacheEfficiencyCompact(pct int) string {
	return fmt.Sprintf("%sC%d%%%s", cacheColor(pct), pct, Reset)
}

func renderCacheEfficiencyLabeled(pct int) string {
	return fmt.Sprintf("%sCache:%s%d%%%s", grey, cacheColor(pct), pct, Reset)
}

func apiRatioColor(pct int) string {
	switch {
	case pct >= WaitHigh:
		return red
	case pct >= WaitMedium:
		return yellow
	default:
		return green
	}
}

func renderAPIRatioCompact(pct int) string {
	return fmt.Sprintf("%sA%d%%%s", apiRatioColor(pct), pct, Reset)
}

func renderAPIRatioLabeled(pct int) string {
	return fmt.Sprintf("%sWait:%s%d%%%s", grey, apiRatioColor(pct), pct, Reset)
}

func renderCostVelocityLabeled(perMin float64) string {
	var color string
	switch {
	case perMin >= CostHigh:
		color = boldRed
	case perMin >= CostMedium:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("%sCost:%s$%.2f/m%s", grey, color, perMin, Reset)
}

func renderResponseSpeed(tokPerSec int) string {
	var color string
	switch {
	case tokPerSec >= SpeedFast:
		color = green // fast
	case tokPerSec >= SpeedModerate:
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
		return color + "N" + Reset
	case "insert":
		color = green
		return color + "I" + Reset
	case "visual":
		color = magenta
		return color + "V" + Reset
	default:
		return ""
	}
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
func renderQuota(u *UsageData) string {
	color5 := quotaColor(u.RemainingPercent5h)
	color7 := quotaColor(u.RemainingPercent7d)

	now := time.Now()
	until5h := formatTimeUntil(now, u.ResetsAt5h)
	until7d := formatTimeUntil(now, u.ResetsAt7d)

	return fmt.Sprintf("%s(%s)5h:%s %s%.0f%%%s/%s%.0f%%%s %s:7d(%s)%s",
		dim, until5h, Reset,
		color5, u.RemainingPercent5h, Reset,
		color7, u.RemainingPercent7d, Reset,
		grey, until7d, Reset)
}

func formatTimeUntil(now, target time.Time) string {
	if target.IsZero() {
		return "?"
	}
	diff := target.Sub(now)
	if diff < 0 {
		return "0"
	}

	hours := int(diff.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	remainHours := hours % 24
	if remainHours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, remainHours)
}

func quotaColor(remaining float64) string {
	switch {
	case remaining < QuotaCritical:
		return boldRed
	case remaining < QuotaLow:
		return red
	case remaining < QuotaMedium:
		return orange
	case remaining < QuotaHigh:
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
