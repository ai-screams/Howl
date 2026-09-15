package internal

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEffortLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"absent", "", ""},
		{"explicit null", "null", ""},
		{"level string", `"xhigh"`, "xhigh"},
		{"numeric token budget", "32000", "32K"},
		{"unsupported shape", `{"level":"high"}`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := effortLabel(json.RawMessage(tt.raw)); got != tt.want {
				t.Errorf("effortLabel(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSubagentName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task SubagentTask
		want string
	}{
		{"prefers name", SubagentTask{Name: "n", Label: "l", Type: "t"}, "n"},
		{"falls back to label", SubagentTask{Label: "l", Type: "t"}, "l"},
		{"then type", SubagentTask{Type: "t", Description: "d"}, "t"},
		{"then description", SubagentTask{Description: "d"}, "d"},
		{"placeholder when bare", SubagentTask{}, "agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := subagentName(tt.task); got != tt.want {
				t.Errorf("subagentName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderSubagentRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		task     SubagentTask
		width    int
		want     []string
		wantGone []string
	}{
		{
			name:  "full row",
			task:  SubagentTask{Name: "explore", Status: "running", TokenCount: 50000, ContextWindowSize: 200000, Effort: json.RawMessage(`"high"`)},
			width: 200,
			want:  []string{"explore", "running", "25%", "50K", "E:high"},
		},
		{
			name:     "token count without a resolved window",
			task:     SubagentTask{Name: "explore", TokenCount: 50000},
			width:    200,
			want:     []string{"explore", "50K"},
			wantGone: []string{"%"},
		},
		{
			name:     "nothing to add beyond the name keeps the default row",
			task:     SubagentTask{Name: "explore"},
			width:    200,
			wantGone: []string{"explore"},
		},
		{
			name:     "narrow width drops trailing segments before wrapping",
			task:     SubagentTask{Name: "explore", Status: "running", TokenCount: 50000, ContextWindowSize: 200000, Effort: json.RawMessage(`"high"`)},
			width:    18,
			want:     []string{"explore"},
			wantGone: []string{"E:high"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripANSI(RenderSubagentRow(tt.task, tt.width, DefaultThresholds()))
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("row %q missing %q", got, w)
				}
			}
			for _, g := range tt.wantGone {
				if strings.Contains(got, g) {
					t.Errorf("row %q should not contain %q", got, g)
				}
			}
			if tt.width > 0 && visibleLen(got) > tt.width {
				t.Errorf("row %q is %d columns, over the %d budget", got, visibleLen(got), tt.width)
			}
		})
	}
}

func TestRenderSubagentRows(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		if rows := RenderSubagentRows(nil, DefaultThresholds()); rows != nil {
			t.Errorf("RenderSubagentRows(nil, DefaultThresholds()) = %v, want nil", rows)
		}
	})

	t.Run("skips tasks without an id", func(t *testing.T) {
		t.Parallel()
		in := &SubagentInput{Columns: 120, Tasks: []SubagentTask{
			{ID: "a", Name: "first", Status: "running"},
			{Name: "unmatched", Status: "running"},
			{ID: "b", Name: "second", Status: "queued"},
		}}
		rows := RenderSubagentRows(in, DefaultThresholds())
		if len(rows) != 2 {
			t.Fatalf("got %d rows, want 2: %+v", len(rows), rows)
		}
		if rows[0].ID != "a" || rows[1].ID != "b" {
			t.Errorf("unexpected ids: %q, %q", rows[0].ID, rows[1].ID)
		}
	})

	t.Run("tolerates undocumented field encodings", func(t *testing.T) {
		t.Parallel()
		// startTime and tokenSamples have no documented encoding, so decoding
		// must not fail whatever shape they arrive in.
		raw := `{"columns":100,"tasks":[{"id":"a","name":"x","status":"running","startTime":"2026-09-15T00:00:00Z","tokenSamples":[1,2,3]},
		                                {"id":"b","name":"y","status":"done","startTime":1789455504,"tokenSamples":null}]}`
		var in SubagentInput
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if len(RenderSubagentRows(&in, DefaultThresholds())) != 2 {
			t.Errorf("want both rows rendered")
		}
	})
}

// If a token count never arrives — a new task, or a field name that changed
// under us — the row must not be less informative than Claude Code's default.
func TestRenderSubagentRow_DescriptionFallback(t *testing.T) {
	t.Parallel()

	t.Run("carries the description when there is no token count", func(t *testing.T) {
		t.Parallel()
		got := stripANSI(RenderSubagentRow(SubagentTask{Name: "explore", Description: "find the config loader"}, 200, DefaultThresholds()))
		if !strings.Contains(got, "find the config loader") {
			t.Errorf("row %q dropped the description with no token count", got)
		}
	})

	t.Run("token count wins over the description", func(t *testing.T) {
		t.Parallel()
		got := stripANSI(RenderSubagentRow(SubagentTask{Name: "explore", Description: "find the config loader", TokenCount: 50000, ContextWindowSize: 200000}, 200, DefaultThresholds()))
		if strings.Contains(got, "find the config loader") {
			t.Errorf("row %q kept the description although it had real numbers", got)
		}
		if !strings.Contains(got, "25%") {
			t.Errorf("row %q lost the context percentage", got)
		}
	})

	t.Run("a description used as the name is not repeated as a segment", func(t *testing.T) {
		t.Parallel()
		// The description becomes the name, leaving nothing else to say, so the
		// default row stands rather than an override that only echoes it.
		if got := RenderSubagentRow(SubagentTask{Description: "only-a-description"}, 200, DefaultThresholds()); got != "" {
			t.Errorf("row = %q, want the default row kept", stripANSI(got))
		}
	})
}

// An override that says only the agent's name is less informative than Claude
// Code's default row, so Howl declines to override in that case.
func TestRenderSubagentRow_DeclinesWhenItAddsNothing(t *testing.T) {
	t.Parallel()

	bare := []SubagentTask{
		{Name: "explore"},
		{Label: "explore"},
		{Type: "Explore"},
		{}, // not even a name
	}
	for _, task := range bare {
		if got := RenderSubagentRow(task, 200, DefaultThresholds()); got != "" {
			t.Errorf("RenderSubagentRow(%+v) = %q, want the default row kept", task, stripANSI(got))
		}
	}

	// One real segment is enough to justify an override.
	withStatus := []SubagentTask{
		{Name: "explore", Status: "running"},
		{Name: "explore", TokenCount: 1000},
		{Name: "explore", Description: "find the loader"},
		{Name: "explore", Effort: json.RawMessage(`"high"`)},
	}
	for _, task := range withStatus {
		if got := RenderSubagentRow(task, 200, DefaultThresholds()); got == "" {
			t.Errorf("RenderSubagentRow(%+v) declined although it had something to add", task)
		}
	}
}

// len() counts bytes while the slice indexes runes. The bug only bites when a
// string is LONGER than the bound in bytes but SHORTER in runes: the guard
// fires, then the rune slice is indexed past its length — padding the result
// with NUL runes when capacity allows, and panicking when it does not. A string
// that is also long in runes slices correctly by accident, so those cases prove
// nothing; every case below is chosen to straddle the bound.
func TestRenderSubagentRow_MultiByteTruncation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task SubagentTask
	}{
		// 10 hangul runes = 30 bytes: over the 24-byte guard, under 24 runes.
		{"name straddles the bound", SubagentTask{Name: strings.Repeat("가", 10), Status: "running"}},
		// 14 runes = 42 bytes, the capacity-32 case that pads with NULs.
		{"name straddles at capacity", SubagentTask{Name: strings.Repeat("가", 14), Status: "running"}},
		// 17 runes = 51 bytes: over the 40-byte guard, under 40 runes.
		{"description straddles the bound", SubagentTask{Name: "탐색", Description: strings.Repeat("설", 17)}},
		{"real korean description", SubagentTask{Name: "탐색", Description: "설정 로더를 찾아서 분석해 주세요"}},
		// Emoji are 4 bytes each, so 7 runes = 28 bytes straddles the name bound.
		{"emoji straddles the bound", SubagentTask{Name: strings.Repeat("🔍", 7), Status: "running"}},
		// Genuinely long inputs must still truncate to the rune bound.
		{"long hangul name truncates", SubagentTask{Name: strings.Repeat("가", 40), Status: "running"}},
		{"long hangul description truncates", SubagentTask{Name: "탐색", Description: strings.Repeat("설", 60)}},
		{"ascii is unaffected", SubagentTask{Name: strings.Repeat("a", 40), Status: "running"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RenderSubagentRow(tt.task, 200, DefaultThresholds()) // must not panic
			if strings.ContainsRune(got, 0) {
				t.Errorf("row contains NUL runes from an over-long slice: %q", got)
			}
			if strings.Contains(got, "\uFFFD") {
				t.Errorf("row contains a replacement rune, so a slice split a character: %q", got)
			}
			// Whatever survives must be a prefix of the input, never padding.
			if n := len([]rune(subagentName(tt.task))); n > 24 && !strings.Contains(stripANSI(got), string([]rune(subagentName(tt.task))[:24])) {
				t.Errorf("truncated name is not a prefix of the input: %q", stripANSI(got))
			}
		})
	}
}

// The shed loop stops at one segment, so the name is the only part that can
// still overrun the row; it must be capped by the budget as well.
func TestRenderSubagentRow_NameFitsNarrowBudget(t *testing.T) {
	t.Parallel()

	for _, width := range []int{4, 8, 16, 24, 80} {
		task := SubagentTask{Name: strings.Repeat("x", 100), Status: "running"}
		got := RenderSubagentRow(task, width, DefaultThresholds())
		if n := visibleLen(got); n > width {
			t.Errorf("width=%d: row is %d columns: %q", width, n, stripANSI(got))
		}
	}

	// With no budget the name still respects its own rune limit.
	got := stripANSI(RenderSubagentRow(SubagentTask{Name: strings.Repeat("x", 100), Status: "running"}, 0, DefaultThresholds()))
	if !strings.HasPrefix(got, strings.Repeat("x", 24)+" ") {
		t.Errorf("unbudgeted row did not cap the name at 24 runes: %q", got)
	}
}
