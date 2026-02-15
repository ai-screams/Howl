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

	// Try to fetch usage quota (optional)
	usage := internal.GetUsage(data.SessionID)

	// Parse transcript for tools/agents (optional)
	toolInfo := internal.ParseTranscript(data.TranscriptPath)

	// Get account info (optional)
	account := internal.GetAccountInfo()

	lines := internal.Render(internal.RenderContext{
		Data:    &data,
		Metrics: metrics,
		Git:     git,
		Usage:   usage,
		Tools:   toolInfo,
		Account: account,
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
