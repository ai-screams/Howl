package internal

import "fmt"

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
// Always 3 lines. Danger mode (85%+) shows extra detail (token breakdown, $/h).
func Render(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo) []string {
	if m.ContextPercent >= DangerThreshold {
		return renderDangerMode(d, m, git, usage, tools)
	}
	return renderNormalMode(d, m, git, usage, tools)
}

func renderNormalMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo) []string {
	// Line 1: [Model] | Bar% (tokens) | $cost | duration
	line1 := make([]string, 0, 5)
	line1 = append(line1, renderModelBadge(d.Model))
	line1 = append(line1, renderContextBar(m.ContextPercent, d.ContextWindow))
	if costStr := renderCost(d.Cost.TotalCostUSD); costStr != "" {
		line1 = append(line1, costStr)
	}
	line1 = append(line1, renderDuration(d.Cost.TotalDurationMS))

	// Line 2: git | code changes | speed | quota (right side)
	line2 := make([]string, 0, 6)
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

func renderDangerMode(d *StdinData, m Metrics, git *GitInfo, usage *UsageData, tools *ToolInfo) []string {
	// Line 1: 🔴 [Model] Bar% (tokens) | $cost | duration
	line1 := make([]string, 0, 4)
	line1 = append(line1, renderModelBadge(d.Model))
	line1 = append(line1, renderContextBar(m.ContextPercent, d.ContextWindow))
	if costStr := renderCost(d.Cost.TotalCostUSD); costStr != "" {
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
	if contains(toLower(m.ID), "anthropic.claude-") {
		suffix = " BR" // Bedrock
	} else if contains(toLower(m.ID), "publishers/anthropic") {
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

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

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

	// Format with K suffix
	usedK := float64(usedTokens) / 1000.0
	totalK := float64(totalTokens) / 1000.0

	return fmt.Sprintf("%s%s%s%s %d%% (%.1fK/%.0fK)", prefix, color, bar, Reset, percent, usedK, totalK)
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

func renderCacheEfficiencyCompact(pct int) string {
	var color string
	switch {
	case pct >= CacheExcellent:
		color = green
	case pct >= CacheGood:
		color = yellow
	default:
		color = red
	}
	return fmt.Sprintf("%sC%d%%%s", color, pct, Reset)
}

func renderCacheEfficiencyLabeled(pct int) string {
	var color string
	switch {
	case pct >= CacheExcellent:
		color = green
	case pct >= CacheGood:
		color = yellow
	default:
		color = red
	}
	return fmt.Sprintf("%sCache:%s%d%%%s", grey, color, pct, Reset)
}

func renderAPIRatioCompact(pct int) string {
	var color string
	switch {
	case pct >= WaitHigh:
		color = red
	case pct >= WaitMedium:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("%sA%d%%%s", color, pct, Reset)
}

func renderAPIRatioLabeled(pct int) string {
	var color string
	switch {
	case pct >= WaitHigh:
		color = red
	case pct >= WaitMedium:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("%sWait:%s%d%%%s", grey, color, pct, Reset)
}

func renderCostVelocity(perMin float64) string {
	var color string
	switch {
	case perMin >= CostHigh:
		color = boldRed
	case perMin >= CostMedium:
		color = yellow
	default:
		color = green
	}
	return fmt.Sprintf("%s$%.2f/m%s", color, perMin, Reset)
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
	m := toLower(mode)
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
	// Take first 8 chars
	if len(name) > 8 {
		name = name[:8]
	}
	return cyan + "@" + name + Reset
}

func renderTokenBreakdown(cu *CurrentUsage) string {
	inK := float64(cu.InputTokens) / 1000.0
	outK := float64(cu.OutputTokens) / 1000.0
	cacheK := float64(cu.CacheReadInputTokens) / 1000.0

	return fmt.Sprintf("%sIn:%.1fK%s %sOut:%.1fK%s %sCache:%.1fK%s",
		grey, inK, Reset,
		grey, outK, Reset,
		green, cacheK, Reset)
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
	// Simple sort
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].count > entries[i].count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
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
