package internal

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSanitizeText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is untouched", "explore", "explore"},
		{"hangul is untouched", "탐색 에이전트", "탐색 에이전트"},
		{"emoji is untouched", "🔍 find", "🔍 find"},
		{"NBSP is untouched", "a b", "a b"},
		{"escape is stripped", "a\033[31mb", "a[31mb"},
		{"OSC 52 clipboard write is stripped", "x\033]52;c;UFdORUQ=\007", "x]52;c;UFdORUQ="},
		{"screen clear is stripped", "x\033[2J", "x[2J"},
		{"NUL is stripped", "a\x00b", "ab"},
		{"newline is stripped", "a\nb", "ab"},
		{"carriage return is stripped", "a\rb", "ab"},
		{"DEL is stripped", "a\x7fb", "ab"},
		{"all control leaves empty", "\033\x00\x07", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeText(tt.in); got != tt.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Howl prints raw ANSI, so an escape carried inside a rendered value is executed
// by the terminal rather than shown. Every value below comes from somewhere the
// user does not control — model-generated names, the git remote, the transcript
// — so none of them may contribute an escape byte to the output.
func TestRender_NoEscapeInjectionFromInput(t *testing.T) {
	evil := "\033]52;c;UFdORUQ=\007\033[2J\033[31mX\r\n"

	// Every string field carries the payload. An earlier version of this test
	// left Version and the workspace dirs clean, and those were exactly the two
	// fields the sanitize sweep missed — the test agreed with the blind spot.
	data := &StdinData{
		Model:       Model{DisplayName: "Opus" + evil, ID: "claude-opus-5"},
		SessionName: "sess" + evil,
		Version:     "2.1.272" + evil,
		Workspace: Workspace{
			CurrentDir: "/tmp/p" + evil,
			ProjectDir: "/tmp/p" + evil,
			Repo:       &RepoInfo{Host: "h", Owner: "own" + evil, Name: "nm" + evil},
			AddedDirs:  []string{"/a"},
		},
		ContextWindow: ContextWindow{UsedPercentage: floatPtr(30), ContextWindowSize: 200000, TotalInputTokens: 60000},
		Cost:          Cost{TotalCostUSD: 1, TotalDurationMS: 60000, TotalAPIDurationMS: 1000},
		OutputStyle:   &OutputStyle{Name: "style" + evil},
		Effort:        &Effort{Level: "high" + evil},
		Agent:         &Agent{Name: "ag" + evil},
		PR:            &PullRequest{Number: 7, URL: "https://x/" + evil, ReviewState: "approved" + evil},
		Worktree:      &Worktree{Name: "wt" + evil},
		PromptCache: &PromptCache{
			CachingObserved: true, Warm: true, ExpiresAt: int64Ptr(time.Now().Add(time.Hour).Unix()),
			HitRatio: floatPtr(0.9), Misses: 1,
			LastMissCause: &LastMissCause{Causes: []string{"cause" + evil}},
		},
	}

	cfg := PresetConfig("full")
	f := &cfg.Features
	f.PromptCache, f.FastMode, f.Exceeds200K, f.OutputStyle, f.Repo, f.AddedDirs = true, true, true, true, true, true
	f.Effort, f.Thinking, f.SessionName, f.PullRequest, f.Worktree, f.AgentName = true, true, true, true, true, true

	rc := RenderContext{
		Data:    data,
		Metrics: ComputeMetrics(data),
		Git:     &GitInfo{Branch: "br" + evil, Dirty: true},
		Account: &AccountInfo{EmailAddress: "e@x" + evil},
		Tools:   &ToolInfo{Tools: map[string]int{"Bash" + evil: 3}, Agents: []string{"agent" + evil}},
		Config:  cfg,
	}

	for _, mode := range []string{"normal", "danger"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "danger" {
				data.ContextWindow.UsedPercentage = floatPtr(95)
				rc.Metrics = ComputeMetrics(data)
			}
			for i, line := range Render(rc) {
				if err := onlyHowlsOwnEscapes(line); err != "" {
					t.Errorf("line %d: %s\nraw: %q", i+1, err, line)
				}
			}
		})
	}
}

// onlyHowlsOwnEscapes reports the first escape sequence in s that Howl does not
// itself emit. Howl emits exactly two shapes: CSI SGR (ESC [ digits/semicolons
// m) and OSC 8 hyperlinks (ESC ] 8 ; ; url BEL). Anything else — an OSC 52
// clipboard write, a screen clear, a cursor move — came from input.
//
// Stripping escapes before checking, as an earlier version of this test did,
// cannot work: the stripper removes injected sequences along with Howl's own,
// so the assertion passes even when sanitizing is removed.
func onlyHowlsOwnEscapes(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			if s[i] < 0x20 || s[i] == 0x7f {
				return "control byte " + string(rune(s[i])) + " outside any escape sequence"
			}
			continue
		}
		if i+1 >= len(s) {
			return "dangling ESC at end of line"
		}
		switch s[i+1] {
		case '[': // CSI SGR
			j := i + 2
			for j < len(s) && (s[j] == ';' || (s[j] >= '0' && s[j] <= '9')) {
				j++
			}
			if j >= len(s) || s[j] != 'm' {
				return "CSI sequence that is not SGR: " + strconv.Quote(s[i:min(i+12, len(s))])
			}
			i = j
		case ']': // OSC — only the hyperlink form is ours
			if !strings.HasPrefix(s[i:], "\033]8;;") {
				return "OSC sequence Howl never emits: " + strconv.Quote(s[i:min(i+16, len(s))])
			}
			j := i + len("\033]8;;")
			for j < len(s) && s[j] != 0x07 {
				if s[j] == 0x1b || s[j] < 0x20 {
					return "control byte inside an OSC 8 payload: " + strconv.Quote(s[i:min(j+2, len(s))])
				}
				j++
			}
			if j >= len(s) {
				return "unterminated OSC 8 sequence"
			}
			i = j
		default:
			return "escape Howl never emits: " + strconv.Quote(s[i:min(i+8, len(s))])
		}
	}
	return ""
}

// buildBar writes width-filled runes in its second loop, so a filled count
// outside [0, width] makes it write without bound. A negative used_percentage
// reached it before: at -100,000,000 one render produced 30MB of output, and
// the status line runs every few seconds.
func TestContextPercent_ClampsBothEnds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pct  float64
		want int
	}{
		{"negative collapses to zero", -100000000, 0},
		{"small negative collapses to zero", -1, 0},
		{"zero", 0, 0},
		{"normal", 42, 42},
		{"over 100 clamps", 100000, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &StdinData{ContextWindow: ContextWindow{UsedPercentage: floatPtr(tt.pct), ContextWindowSize: 200000}}
			if got := calcContextPercent(d); got != tt.want {
				t.Errorf("calcContextPercent(%v) = %d, want %d", tt.pct, got, tt.want)
			}
		})
	}

	t.Run("fallback path clamps too", func(t *testing.T) {
		t.Parallel()
		d := &StdinData{ContextWindow: ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage:      &CurrentUsage{InputTokens: -100000000},
		}}
		if got := calcContextPercent(d); got != 0 {
			t.Errorf("calcContextPercent(negative tokens) = %d, want 0", got)
		}
	})
}

func TestBuildBar_BoundsAreEnforced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filled, width int
		wantRunes     int
	}{
		{"normal", 4, 10, 10},
		{"full", 10, 10, 10},
		{"empty", 0, 10, 10},
		{"negative filled", -1000000, 10, 10},
		{"filled over width", 1000000, 10, 10},
		{"zero width", 5, 0, 0},
		{"negative width", 5, -10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildBar(tt.filled, tt.width)
			if n := len([]rune(got)); n != tt.wantRunes {
				t.Errorf("buildBar(%d, %d) produced %d runes, want %d", tt.filled, tt.width, n, tt.wantRunes)
			}
		})
	}
}

// A status line has no reason to hand the terminal a javascript: or file:
// target, and emulators differ on which schemes they will open.
func TestOSC8Link_OnlyWebSchemes(t *testing.T) {
	t.Parallel()

	linked := []string{"https://github.com/o/r/pull/1", "http://example.org/x"}
	for _, u := range linked {
		if got := osc8Link(u, "PR#1"); !strings.Contains(got, u) {
			t.Errorf("osc8Link(%q) did not link a web URL: %q", u, got)
		}
	}

	refused := []string{"javascript:alert(1)", "file:///etc/passwd", "data:text/html,x", "vscode://x", "", "//evil", "ftp://x"}
	for _, u := range refused {
		got := osc8Link(u, "PR#1")
		if got != "PR#1" {
			t.Errorf("osc8Link(%q) = %q, want the bare text with no hyperlink", u, got)
		}
	}

	// A crafted URL cannot close the sequence early or start a nested one: with
	// ESC and BEL stripped, what is left is inert text inside the URI.
	got := osc8Link("https://x/\a\033]52;c;X\a", "PR#1")
	if payload, _, ok := strings.Cut(strings.TrimPrefix(got, "\033]8;;"), "\a"); !ok {
		t.Fatalf("hyperlink is not BEL-terminated: %q", got)
	} else if strings.ContainsAny(payload, "\x1b\x07") {
		t.Errorf("URL payload still carries a control byte: %q", payload)
	}
}
