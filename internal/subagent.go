package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SubagentInput is the JSON object Claude Code pipes to a subagentStatusLine
// command. It carries the common hook fields (ignored here), the usable row
// width, and one entry per visible subagent row.
//
// Unlike the main statusline schema, these field names are camelCase.
type SubagentInput struct {
	Columns int            `json:"columns"`
	Tasks   []SubagentTask `json:"tasks"`
}

// SubagentTask is one row in the agent panel. Availability varies by Claude
// Code version: Model and ContextWindowSize require 2.1.205+ and are omitted
// until the task's model resolves; Effort requires 2.1.214+ and is absent when
// the subagent inherits the session's effort level.
type SubagentTask struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Label       string `json:"label"`
	CWD         string `json:"cwd"`

	Model             string `json:"model"`
	ContextWindowSize int    `json:"contextWindowSize"`
	TokenCount        int    `json:"tokenCount"`

	// Effort is documented as either an effort level string ("low", "medium",
	// "high", "xhigh", "max") or a numeric token budget, so it is decoded raw
	// and narrowed by effortLabel.
	Effort json.RawMessage `json:"effort"`

	// StartTime and TokenSamples are accepted but not rendered: their encodings
	// are not documented, and Howl does not display values it cannot verify.
	StartTime    json.RawMessage `json:"startTime"`
	TokenSamples json.RawMessage `json:"tokenSamples"`
}

// SubagentRow is one line of the command's stdout: the row body Claude Code
// substitutes for the task named by ID.
type SubagentRow struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// effortLabel narrows the string-or-number effort field. Returns "" when the
// field is absent or neither shape.
func effortLabel(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return formatTokenCount(int(n))
	}
	return ""
}

// Row text limits, in runes.
const (
	maxNameRunes = 24
	maxDescRunes = 40
)

// subagentName picks the most specific label Claude Code supplied for a row.
func subagentName(t SubagentTask) string {
	for _, c := range []string{t.Name, t.Label, t.Type, t.Description} {
		if c := sanitizeText(c); c != "" {
			return c
		}
	}
	return "agent"
}

// RenderSubagentRow builds the row body for one task, or "" to leave Claude
// Code's default rendering in place. Every segment past the name is omitted
// when its source is absent, matching the main statusline's nil-means-skip rule.
func RenderSubagentRow(t SubagentTask, maxWidth int, th Thresholds) string {
	// fullName is kept untruncated so the description check below still
	// recognises a description that was promoted to be the name.
	fullName := subagentName(t)
	name := fullName
	// The shed loop below stops at one segment, so the name is the one part
	// that can still overrun the row. Cap it by the budget as well as by its
	// own limit. Count runes, not bytes: len() on a multi-byte name would slice
	// past the rune slice's capacity and panic, or pad the result with NULs.
	nameBudget := maxNameRunes
	if maxWidth > 0 && maxWidth < nameBudget {
		nameBudget = maxWidth
	}
	if runes := []rune(name); len(runes) > nameBudget {
		name = string(runes[:nameBudget])
	}

	segs := []string{cyan + name + Reset}

	if status := sanitizeText(t.Status); status != "" {
		segs = append(segs, grey+status+Reset)
	}

	// Claude Code's default row is "name · description · token count". When no
	// token count arrives — the task is new, or a field name changed under us —
	// carry the description so the override degrades to roughly the default
	// instead of to a bare name.
	if desc := sanitizeText(t.Description); t.TokenCount == 0 && desc != "" && desc != fullName {
		if runes := []rune(desc); len(runes) > maxDescRunes {
			desc = string(runes[:maxDescRunes])
		}
		segs = append(segs, grey+desc+Reset)
	}

	// Per-row context usage: tokenCount against that model's own window.
	if t.ContextWindowSize > 0 && t.TokenCount > 0 {
		pct := t.TokenCount * 100 / t.ContextWindowSize
		segs = append(segs, fmt.Sprintf("%s%d%%%s %s(%s)%s",
			contextColor(pct, th), pct, Reset,
			grey, formatTokenCount(t.TokenCount), Reset))
	} else if t.TokenCount > 0 {
		segs = append(segs, grey+formatTokenCount(t.TokenCount)+Reset)
	}

	if e := sanitizeText(effortLabel(t.Effort)); e != "" {
		segs = append(segs, dim+"E:"+e+Reset)
	}

	out := strings.Join(segs, grey+" · "+Reset)

	// Drop trailing segments rather than let the row wrap.
	for len(segs) > 1 && maxWidth > 0 && visibleLen(out) > maxWidth {
		segs = segs[:len(segs)-1]
		out = strings.Join(segs, grey+" · "+Reset)
	}

	// Claude Code's default row is "name · description · token count". With
	// nothing but the name left to show — either because there was nothing
	// else, or because the width budget shed the rest — an override would be
	// strictly less informative, so leave the row alone. This has to run after
	// the shed loop: checking before it would let a trimmed row through.
	if len(segs) == 1 {
		return ""
	}
	return out
}

// RenderSubagentRows renders every task that Howl has something better to say
// about. Tasks it would render identically to the default are left out, so
// Claude Code keeps its own row for them.
func RenderSubagentRows(in *SubagentInput, th Thresholds) []SubagentRow {
	if in == nil {
		return nil
	}
	rows := make([]SubagentRow, 0, len(in.Tasks))
	for _, t := range in.Tasks {
		if t.ID == "" {
			continue // without an id Claude Code cannot match the override
		}
		if c := RenderSubagentRow(t, in.Columns, th); c != "" {
			rows = append(rows, SubagentRow{ID: t.ID, Content: c})
		}
	}
	return rows
}
