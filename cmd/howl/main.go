// Package main provides the Howl statusline HUD binary for Claude Code.
// It reads JSON from stdin, computes metrics, and outputs ANSI-formatted statuslines.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ai-screams/howl/internal"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version":
			fmt.Printf("howl %s (%s)\n", version, commit)
			os.Exit(0)
		case "-h", "--help":
			fmt.Fprintln(os.Stderr, "howl: Claude Code statusline HUD. Reads JSON from stdin.")
			fmt.Fprintln(os.Stderr, "  --subagent  render subagent panel rows (settings key: subagentStatusLine)")
			os.Exit(0)
		case "--subagent":
			runSubagent()
			os.Exit(0)
		}
	}

	var data internal.StdinData
	if err := json.NewDecoder(os.Stdin).Decode(&data); err != nil {
		fmt.Fprint(os.Stderr, "howl: stdin parse error")
		os.Exit(1)
	}

	cfg := internal.LoadConfig()
	metrics := internal.ComputeMetrics(&data)

	dir := data.Workspace.ProjectDir
	if dir == "" {
		dir = data.CWD
	}
	git := internal.GetGitInfo(dir)

	// Quota comes directly from stdin rate_limits (optional, subscriber-only)
	usage := internal.UsageFromRateLimits(data.RateLimits)

	// Parse transcript for tools/agents (optional)
	toolInfo := internal.ParseTranscript(data.TranscriptPath)

	// Get account info (optional)
	account := internal.GetAccountInfo()

	// A newer release, if the plugin's session-start hook recorded one.
	update := internal.ReadUpdateNotice(version)

	lines := internal.Render(internal.RenderContext{
		Data:    &data,
		Metrics: metrics,
		Git:     git,
		Usage:   usage,
		Tools:   toolInfo,
		Account: account,
		Update:  update,
		Config:  cfg,
	})

	// Output each line individually with:
	// 1. RESET prefix to clear ANSI state
	// 2. Spaces replaced with NBSP (\u00A0) to prevent Claude Code from stripping them
	for _, line := range lines {
		line = strings.ReplaceAll(line, " ", "\u00A0")
		fmt.Println(internal.Reset + line)
	}
}

// runSubagent serves the subagentStatusLine setting: it reads the agent panel's
// task list and writes one JSON line per row it wants to override. A row Howl
// says nothing about is left out so Claude Code keeps its default rendering.
func runSubagent() {
	var in internal.SubagentInput
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		fmt.Fprint(os.Stderr, "howl: subagent stdin parse error")
		os.Exit(1)
	}

	// The agent panel honours the same configured thresholds as the main line.
	cfg := internal.LoadConfig()

	enc := json.NewEncoder(os.Stdout)
	for _, row := range internal.RenderSubagentRows(&in, cfg.Thresholds) {
		// The docs say row content renders as-is, so unlike the main status
		// line there is no documented space stripping here. NBSP is applied
		// anyway as a precaution: it occupies one column and renders as a
		// space, so it is correct either way.
		row.Content = strings.ReplaceAll(row.Content, " ", "\u00A0")
		if err := enc.Encode(row); err != nil {
			return
		}
	}
}
