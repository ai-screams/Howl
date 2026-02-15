package internal

import (
	"strings"
	"testing"
)

func TestRenderModelBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		model      Model
		wantColor  string
		wantName   string
		wantSuffix string
	}{
		{
			name:      "opus gets gold color",
			model:     Model{DisplayName: "Claude Opus 4"},
			wantColor: boldYlw,
			wantName:  "Claude Opus 4",
		},
		{
			name:      "sonnet gets cyan color",
			model:     Model{DisplayName: "Claude Sonnet 4.5"},
			wantColor: cyan,
			wantName:  "Claude Sonnet 4.5",
		},
		{
			name:      "haiku gets green color",
			model:     Model{DisplayName: "Claude Haiku 4.5"},
			wantColor: green,
			wantName:  "Claude Haiku 4.5",
		},
		{
			name:      "unknown gets dim color",
			model:     Model{DisplayName: "GPT-4"},
			wantColor: dim,
			wantName:  "GPT-4",
		},
		{
			name:      "empty DisplayName falls back to ID",
			model:     Model{ID: "claude-opus-4"},
			wantColor: boldYlw,
			wantName:  "claude-opus-4",
		},
		{
			name:      "both empty shows question mark",
			model:     Model{},
			wantColor: dim,
			wantName:  "?",
		},
		{
			name:       "bedrock ID gets BR suffix",
			model:      Model{ID: "anthropic.claude-sonnet-3-5", DisplayName: "Sonnet"},
			wantColor:  cyan,
			wantSuffix: " BR",
		},
		{
			name:       "vertex ID gets VX suffix",
			model:      Model{ID: "publishers/anthropic/models/sonnet", DisplayName: "Sonnet"},
			wantColor:  cyan,
			wantSuffix: " VX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderModelBadge(tt.model, 0)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderModelBadge(%v) missing color %q in %q", tt.model, tt.wantColor, got)
			}
			if tt.wantName != "" && !strings.Contains(got, tt.wantName) {
				t.Errorf("renderModelBadge(%v) missing name %q in %q", tt.model, tt.wantName, got)
			}
			if tt.wantSuffix != "" && !strings.Contains(got, tt.wantSuffix) {
				t.Errorf("renderModelBadge(%v) missing suffix %q in %q", tt.model, tt.wantSuffix, got)
			}
			if !strings.Contains(got, "[") || !strings.Contains(got, "]") {
				t.Errorf("renderModelBadge(%v) missing brackets in %q", tt.model, got)
			}
		})
	}
}

func TestContextColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		percent int
		want    string
	}{
		{"85 is boldRed", 85, boldRed},
		{"90 is boldRed", 90, boldRed},
		{"100 is boldRed", 100, boldRed},
		{"84 is orange", 84, orange},
		{"70 is orange", 70, orange},
		{"69 is yellow", 69, yellow},
		{"50 is yellow", 50, yellow},
		{"49 is green", 49, green},
		{"0 is green", 0, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contextColor(tt.percent, DefaultThresholds())
			if got != tt.want {
				t.Errorf("contextColor(%d) = %q, want %q", tt.percent, got, tt.want)
			}
		})
	}
}

func TestRenderCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		usd       float64
		wantEmpty bool
		wantColor string
		wantText  string
	}{
		{"below threshold returns empty", 0.0005, true, "", ""},
		{"zero returns empty", 0, true, "", ""},
		{"small cost default", 0.50, false, Reset, "$0.50"},
		{"medium cost yellow", 1.50, false, yellow, "$1.50"},
		{"high cost boldRed", 5.50, false, boldRed, "$5.50"},
		{">=10 uses one decimal", 15.0, false, boldRed, "$15.0"},
		{"exactly 1.0 is yellow", 1.0, false, yellow, "$1.00"},
		{"exactly 5.0 is boldRed", 5.0, false, boldRed, "$5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCost(tt.usd, DefaultThresholds())
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderCost(%f) = %q, want empty", tt.usd, got)
				}
				return
			}
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCost(%f) missing color %q in %q", tt.usd, tt.wantColor, got)
			}
			if !strings.Contains(got, tt.wantText) {
				t.Errorf("renderCost(%f) missing text %q in %q", tt.usd, tt.wantText, got)
			}
		})
	}
}

func TestRenderDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"under 1 minute", 30000, grey + "<1m" + Reset},
		{"exactly 0", 0, grey + "<1m" + Reset},
		{"exactly 1 minute", 60000, "1m"},
		{"30 minutes", 1800000, "30m"},
		{"exactly 1 hour", 3600000, "1h0m"},
		{"1h32m", 5520000, "1h32m"},
		{"2h0m", 7200000, "2h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderDuration(tt.ms)
			if got != tt.want {
				t.Errorf("renderDuration(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestRenderWorkspace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data *StdinData
		want string
	}{
		{
			name: "uses ProjectDir",
			data: &StdinData{Workspace: Workspace{ProjectDir: "/home/user/project"}},
			want: grey + "project/" + Reset,
		},
		{
			name: "falls back to CurrentDir",
			data: &StdinData{Workspace: Workspace{CurrentDir: "/home/user/myapp"}},
			want: grey + "myapp/" + Reset,
		},
		{
			name: "both empty returns empty",
			data: &StdinData{},
			want: "",
		},
		{
			name: "root path returns empty",
			data: &StdinData{Workspace: Workspace{ProjectDir: "/"}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderWorkspace(tt.data)
			if got != tt.want {
				t.Errorf("renderWorkspace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderGitCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		git  *GitInfo
		want string
	}{
		{
			name: "clean branch",
			git:  &GitInfo{Branch: "main", Dirty: false},
			want: magenta + "main" + Reset,
		},
		{
			name: "dirty branch",
			git:  &GitInfo{Branch: "main", Dirty: true},
			want: magenta + "main*" + Reset,
		},
		{
			name: "empty branch",
			git:  &GitInfo{Branch: ""},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderGitCompact(tt.git)
			if got != tt.want {
				t.Errorf("renderGitCompact(%v) = %q, want %q", tt.git, got, tt.want)
			}
		})
	}
}

func TestRenderLineChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cost      Cost
		wantEmpty bool
		wantAdd   string
		wantDel   string
	}{
		{
			name:      "both zero",
			cost:      Cost{},
			wantEmpty: true,
		},
		{
			name:    "normal changes",
			cost:    Cost{TotalLinesAdded: 150, TotalLinesRemoved: 12},
			wantAdd: "+150",
			wantDel: "-12",
		},
		{
			name:    "large added uses K suffix",
			cost:    Cost{TotalLinesAdded: 1500, TotalLinesRemoved: 0},
			wantAdd: "+1.5K",
			wantDel: "-0",
		},
		{
			name:    "only removed",
			cost:    Cost{TotalLinesAdded: 0, TotalLinesRemoved: 50},
			wantAdd: "+0",
			wantDel: "-50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderLineChanges(tt.cost)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderLineChanges(%v) = %q, want empty", tt.cost, got)
				}
				return
			}
			if !strings.Contains(got, tt.wantAdd) {
				t.Errorf("renderLineChanges(%v) missing %q in %q", tt.cost, tt.wantAdd, got)
			}
			if !strings.Contains(got, tt.wantDel) {
				t.Errorf("renderLineChanges(%v) missing %q in %q", tt.cost, tt.wantDel, got)
			}
		})
	}
}

func TestFormatCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want string
	}{
		{"zero", 0, "0"},
		{"small number", 42, "42"},
		{"999", 999, "999"},
		{"1000", 1000, "1.0K"},
		{"1500", 1500, "1.5K"},
		{"10000", 10000, "10.0K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCount(tt.n)
			if got != tt.want {
				t.Errorf("formatCount(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestFormatTokenCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{"zero", 0, "0K"},
		{"1K tokens", 1000, "1K"},
		{"50K tokens", 50000, "50K"},
		{"200K tokens", 200000, "200K"},
		{"500K tokens", 500000, "500K"},
		{"999K tokens", 999000, "999K"},
		{"1M exact", 1000000, "1M"},
		{"1.5M tokens", 1500000, "1.5M"},
		{"2M exact", 2000000, "2M"},
		{"128K tokens", 128000, "128K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenCount(tt.tokens)
			if got != tt.want {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestRenderAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		acc         *AccountInfo
		wantContain string
	}{
		{
			name:        "full email",
			acc:         &AccountInfo{EmailAddress: "user@example.com"},
			wantContain: "user@example.com",
		},
		{
			name:        "email with display name",
			acc:         &AccountInfo{EmailAddress: "test@test.com", DisplayName: "Test User"},
			wantContain: "test@test.com",
		},
		{
			name:        "empty email returns grey+reset only",
			acc:         &AccountInfo{EmailAddress: ""},
			wantContain: grey + Reset, // no email between color codes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderAccount(tt.acc)
			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("renderAccount() = %q, want to contain %q", got, tt.wantContain)
			}
			// Verify grey color is used for all cases
			if !strings.Contains(got, grey) {
				t.Errorf("renderAccount() should use grey color, got %q", got)
			}
		})
	}
}

func TestRenderVimCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode string
		want string
	}{
		{"normal", "normal", blue + "Normal" + Reset},
		{"NORMAL uppercase", "NORMAL", blue + "Normal" + Reset},
		{"insert", "insert", green + "Insert" + Reset},
		{"visual", "visual", magenta + "Visual" + Reset},
		{"unknown mode", "unknown", ""},
		{"empty mode", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderVimCompact(tt.mode)
			if got != tt.want {
				t.Errorf("renderVimCompact(%q) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestRenderAgentCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{"short name", "foo", cyan + "@foo" + Reset},
		{"exactly 8 chars", "abcdefgh", cyan + "@abcdefgh" + Reset},
		{"long name truncated", "my-long-agent-name", cyan + "@my-long-" + Reset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAgentCompact(tt.agent)
			if got != tt.want {
				t.Errorf("renderAgentCompact(%q) = %q, want %q", tt.agent, got, tt.want)
			}
		})
	}
}

func TestRenderCacheEfficiencyCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pct       int
		wantColor string
	}{
		{"excellent >=80 green", 80, green},
		{"good >=50 yellow", 50, yellow},
		{"poor <50 red", 49, red},
		{"high value", 95, green},
		{"zero", 0, red},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCacheEfficiencyCompact(tt.pct, DefaultThresholds())
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCacheEfficiencyCompact(%d) missing color %q in %q", tt.pct, tt.wantColor, got)
			}
			if !strings.Contains(got, "C") {
				t.Errorf("renderCacheEfficiencyCompact(%d) missing 'C' prefix in %q", tt.pct, got)
			}
		})
	}
}

func TestRenderCacheEfficiencyLabeled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pct       int
		cu        *CurrentUsage
		wantColor string
		wantW     bool
	}{
		{"excellent green", 80, nil, green, false},
		{"good yellow", 50, nil, yellow, false},
		{"poor red", 49, nil, red, false},
		{"with token breakdown", 90, &CurrentUsage{
			CacheCreationInputTokens: 3000,
			CacheReadInputTokens:     120000,
		}, green, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCacheEfficiencyLabeled(tt.pct, DefaultThresholds(), tt.cu)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCacheEfficiencyLabeled(%d) missing color in %q", tt.pct, got)
			}
			if !strings.Contains(got, "Cache:") {
				t.Errorf("renderCacheEfficiencyLabeled(%d) missing 'Cache:' label in %q", tt.pct, got)
			}
			if tt.wantW {
				if !strings.Contains(got, "W:") || !strings.Contains(got, "R:") {
					t.Errorf("renderCacheEfficiencyLabeled(%d) missing W:/R: breakdown in %q", tt.pct, got)
				}
			}
		})
	}
}

func TestRenderAPIRatioLabeled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pct       int
		wantColor string
	}{
		{"high red", 60, red},
		{"medium yellow", 35, yellow},
		{"low green", 34, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAPIRatioLabeled(tt.pct, DefaultThresholds())
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderAPIRatioLabeled(%d) missing color in %q", tt.pct, got)
			}
			if !strings.Contains(got, "Wait:") {
				t.Errorf("renderAPIRatioLabeled(%d) missing 'Wait:' label in %q", tt.pct, got)
			}
		})
	}
}

func TestRenderResponseSpeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		speed     int
		wantColor string
	}{
		{"fast >=60 green", 60, green},
		{"moderate >=30 yellow", 30, yellow},
		{"slow <30 orange", 29, orange},
		{"very fast", 100, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderResponseSpeed(tt.speed, DefaultThresholds())
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderResponseSpeed(%d) missing color %q in %q", tt.speed, tt.wantColor, got)
			}
			if !strings.Contains(got, "tok/s") {
				t.Errorf("renderResponseSpeed(%d) missing 'tok/s' in %q", tt.speed, got)
			}
		})
	}
}

func TestRenderCostVelocityLabeled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		perMin    float64
		wantColor string
	}{
		{"high boldRed", 0.50, boldRed},
		{"medium yellow", 0.10, yellow},
		{"low green", 0.09, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCostVelocityLabeled(tt.perMin, DefaultThresholds())
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCostVelocityLabeled(%f) missing color in %q", tt.perMin, got)
			}
			if !strings.Contains(got, "Cost:") {
				t.Errorf("renderCostVelocityLabeled(%f) missing 'Cost:' in %q", tt.perMin, got)
			}
		})
	}
}

func TestRenderTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tools     map[string]int
		maxWidth  int
		wantEmpty bool
		wantParts []string
		wantPlus  bool // expect "+N" overflow indicator
	}{
		{"nil map", nil, 80, true, nil, false},
		{"empty map", map[string]int{}, 80, true, nil, false},
		{"single tool", map[string]int{"Read": 5}, 80, false, []string{"Read", "(5)"}, false},
		{
			"more than 5 tools keeps top 5",
			map[string]int{"A": 1, "B": 2, "C": 3, "D": 4, "E": 5, "F": 6, "G": 7},
			80,
			false,
			[]string{"G", "(7)", "F", "(6)"},
			false,
		},
		{
			"long name truncated",
			map[string]int{"search_for_pattern": 5},
			80,
			false,
			[]string{"search_for_…", "(5)"},
			false,
		},
		{
			"budget overflow shows +N",
			map[string]int{"Read": 10, "Edit": 8, "Bash": 5, "Write": 3, "Grep": 2},
			30,
			false,
			[]string{"Read", "(10)"},
			true,
		},
		{
			"first tool always shown even if exceeds budget",
			map[string]int{"find_symbol": 12},
			5,
			false,
			[]string{"find_symbol", "(12)"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTools(tt.tools, tt.maxWidth)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderTools(%v, %d) = %q, want empty", tt.tools, tt.maxWidth, got)
				}
				return
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("renderTools(%v, %d) missing %q in %q", tt.tools, tt.maxWidth, part, got)
				}
			}
			if tt.wantPlus && !strings.Contains(got, "+") {
				t.Errorf("renderTools(%v, %d) expected +N overflow indicator in %q", tt.tools, tt.maxWidth, got)
			}
		})
	}
}

func TestVisibleLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain text", "hello", 5},
		{"with ANSI color", blue + "hello" + Reset, 5},
		{"multiple ANSI", blue + "a" + Reset + " " + yellow + "b" + Reset, 3},
		{"empty string", "", 0},
		{"ANSI only", blue + Reset, 0},
		{"with parentheses", blue + "Read" + Reset + "(5)", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := visibleLen(tt.input)
			if got != tt.want {
				t.Errorf("visibleLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateToolName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short name unchanged", "Read", 12, "Read"},
		{"exact length unchanged", "find_symbol_", 12, "find_symbol_"},
		{"long name truncated", "search_for_pattern", 12, "search_for_…"},
		{"resolve-library-id", "resolve-library-id", 12, "resolve-lib…"},
		{"maxLen 5", "long_name", 5, "long…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateToolName(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateToolName(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestRenderAgents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agents    []string
		wantEmpty bool
		wantParts []string
		wantMiss  []string
	}{
		{"nil", nil, true, nil, nil},
		{"empty", []string{}, true, nil, nil},
		{"single agent", []string{"code-writer"}, false, []string{"code-writer"}, nil},
		{"three agents shows only first 2", []string{"a", "b", "c"}, false, []string{"a", "b"}, []string{"c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAgents(tt.agents)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderAgents(%v) = %q, want empty", tt.agents, got)
				}
				return
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("renderAgents(%v) missing %q in %q", tt.agents, part, got)
				}
			}
			for _, miss := range tt.wantMiss {
				if strings.Contains(got, miss) {
					t.Errorf("renderAgents(%v) should not contain %q in %q", tt.agents, miss, got)
				}
			}
		})
	}
}

func TestJoinParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"empty", []string{}, ""},
		{"single part", []string{"hello"}, "hello"},
		{"two parts", []string{"a", "b"}, "a " + grey + "|" + Reset + " b"},
		{"three parts", []string{"a", "b", "c"}, "a " + grey + "|" + Reset + " b " + grey + "|" + Reset + " c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinParts(tt.parts)
			if got != tt.want {
				t.Errorf("joinParts(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("normal mode under 85%", func(t *testing.T) {
		d := &StdinData{
			Model:         Model{DisplayName: "Claude Sonnet 4.5"},
			ContextWindow: ContextWindow{ContextWindowSize: 200000},
			Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5, TotalLinesAdded: 100},
			Workspace:     Workspace{CurrentDir: "/test"},
		}
		m := Metrics{ContextPercent: 50}
		// Provide data so full preset shows multiple lines
		git := &GitInfo{Branch: "main"}
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Config: DefaultConfig()})
		if len(lines) < 2 {
			t.Fatalf("Render() returned %d lines, want >= 2", len(lines))
		}
		if !strings.Contains(lines[0], "Sonnet") {
			t.Errorf("normal mode line 1 should contain model name, got %q", lines[0])
		}
	})

	t.Run("danger mode at 85%", func(t *testing.T) {
		d := &StdinData{
			Model:         Model{DisplayName: "Claude Opus 4"},
			ContextWindow: ContextWindow{ContextWindowSize: 200000},
			Cost:          Cost{TotalDurationMS: 300000, TotalCostUSD: 15.0},
		}
		m := Metrics{ContextPercent: 87}
		lines := Render(RenderContext{Data: d, Metrics: m, Config: DefaultConfig()})
		if len(lines) != 2 {
			t.Fatalf("danger mode Render() returned %d lines, want 2", len(lines))
		}
		if !strings.Contains(lines[0], "Opus") {
			t.Errorf("danger mode line 1 should contain model name, got %q", lines[0])
		}
	})

	t.Run("danger mode context bar has danger indicator", func(t *testing.T) {
		d := &StdinData{
			Model:         Model{DisplayName: "Sonnet"},
			ContextWindow: ContextWindow{ContextWindowSize: 200000},
			Cost:          Cost{TotalDurationMS: 60000},
		}
		m := Metrics{ContextPercent: 90}
		lines := Render(RenderContext{Data: d, Metrics: m, Config: DefaultConfig()})
		// Context bar in danger mode should contain the red circle emoji
		found := false
		for _, line := range lines {
			if strings.Contains(line, "🔴") {
				found = true
				break
			}
		}
		if !found {
			t.Error("danger mode should contain 🔴 indicator")
		}
	})

	t.Run("normal mode with git info", func(t *testing.T) {
		d := &StdinData{
			Model:         Model{DisplayName: "Sonnet"},
			ContextWindow: ContextWindow{ContextWindowSize: 200000},
			Cost:          Cost{TotalDurationMS: 60000},
		}
		m := Metrics{ContextPercent: 30}
		git := &GitInfo{Branch: "main", Dirty: true}
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Config: DefaultConfig()})
		found := false
		for _, line := range lines {
			if strings.Contains(line, "main*") {
				found = true
				break
			}
		}
		if !found {
			t.Error("normal mode should show git branch in output")
		}
	})
}

func TestRenderWithConfig(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
	}
	m := Metrics{ContextPercent: 30}
	git := &GitInfo{Branch: "main", Dirty: true}
	account := &AccountInfo{EmailAddress: "user@example.com"}

	t.Run("full preset shows all features", func(t *testing.T) {
		cfg := PresetConfig("full")
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Account: account, Config: cfg})
		// full preset without usage: L1 (model+context bar+account+git+cost+duration) = 1 line
		if len(lines) < 1 {
			t.Errorf("full preset should show 1+ lines, got %d", len(lines))
		}
		// Should contain account email
		output := strings.Join(lines, " ")
		if !strings.Contains(output, "user@example.com") {
			t.Error("full preset should show account email")
		}
		if !strings.Contains(output, "main*") {
			t.Error("full preset should show git branch")
		}
	})

	t.Run("minimal preset shows single line", func(t *testing.T) {
		cfg := PresetConfig("minimal")
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Account: account, Config: cfg})
		// minimal preset without usage: context bar inlined into L1, no L3/L4
		if len(lines) != 1 {
			t.Errorf("minimal preset should show exactly 1 line, got %d", len(lines))
		}
		// Should NOT contain account or git
		output := strings.Join(lines, " ")
		if strings.Contains(output, "user@example.com") {
			t.Error("minimal preset should not show account")
		}
		if strings.Contains(output, "main") {
			t.Error("minimal preset should not show git branch")
		}
		// Should contain context bar (inlined)
		if !strings.Contains(output, "30%") {
			t.Error("minimal preset should show context percent inline")
		}
	})

	t.Run("developer preset shows account and git", func(t *testing.T) {
		cfg := PresetConfig("developer")
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Account: account, Config: cfg})
		// developer without usage: L1 has context bar inlined
		if len(lines) < 1 {
			t.Errorf("developer preset should show 1+ lines, got %d", len(lines))
		}
		output := strings.Join(lines, " ")
		if !strings.Contains(output, "user@example.com") {
			t.Error("developer preset should show account")
		}
		if !strings.Contains(output, "main*") {
			t.Error("developer preset should show git branch")
		}
	})

	t.Run("cost-focused preset shows quota metrics", func(t *testing.T) {
		cfg := PresetConfig("cost-focused")
		usage := &UsageData{RemainingPercent5h: 80.0, RemainingPercent7d: 90.0}
		lines := Render(RenderContext{Data: d, Metrics: m, Usage: usage, Account: account, Config: cfg})
		// cost-focused with usage: L1 (model+context size+cost+dur) + L2 (3 bars) = 2+ lines
		if len(lines) < 2 {
			t.Errorf("cost-focused preset should show 2+ lines, got %d", len(lines))
		}
		// Should contain account (now enabled) but NOT git
		output := strings.Join(lines, " ")
		if !strings.Contains(output, "user@example.com") {
			t.Error("cost-focused preset should show account")
		}
	})
}

func TestRenderDangerModeIgnoresConfig(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
	}
	m := Metrics{ContextPercent: 87} // Danger mode
	git := &GitInfo{Branch: "main", Dirty: true}
	account := &AccountInfo{EmailAddress: "user@example.com"}

	t.Run("minimal preset ignored in danger mode", func(t *testing.T) {
		cfg := PresetConfig("minimal")
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Account: account, Config: cfg})
		// Danger mode always shows 2 lines, ignoring preset
		if len(lines) != 2 {
			t.Errorf("danger mode should show 2 lines regardless of preset, got %d", len(lines))
		}
		// Danger mode should contain danger indicator
		output := strings.Join(lines, " ")
		if !strings.Contains(output, "🔴") {
			t.Error("danger mode should show danger indicator")
		}
	})
}

func TestRenderContextBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		percent int
		cwSize  int
		wantIn  []string
	}{
		{
			name:    "low percentage no indicator",
			percent: 10,
			cwSize:  200000,
			wantIn:  []string{"10%", "█", "░"},
		},
		{
			name:    "warning shows caution",
			percent: 72,
			cwSize:  200000,
			wantIn:  []string{"72%", "⚠"},
		},
		{
			name:    "danger shows red circle",
			percent: 90,
			cwSize:  200000,
			wantIn:  []string{"90%", "🔴"},
		},
		{
			name:    "100 percent all filled",
			percent: 100,
			cwSize:  200000,
			wantIn:  []string{"100%"},
		},
		{
			name:    "1M context shows M suffix",
			percent: 50,
			cwSize:  1000000,
			wantIn:  []string{"50%", "500K/1M"},
		},
		{
			name:    "1M context at 87%",
			percent: 87,
			cwSize:  1000000,
			wantIn:  []string{"87%", "870K/1M"},
		},
		{
			name:    "128K context",
			percent: 75,
			cwSize:  128000,
			wantIn:  []string{"75%", "96K/128K"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cw := ContextWindow{ContextWindowSize: tt.cwSize}
			got := renderContextBar(tt.percent, cw, DefaultThresholds())
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("renderContextBar(%d, %d) missing %q in %q", tt.percent, tt.cwSize, want, got)
				}
			}
		})
	}
}

// ====== New Tests for Coverage Gaps ======

// renderNormalMode coverage improvements

func TestRenderNormalMode_WithTools(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
	}
	m := Metrics{ContextPercent: 50} // Under 85% = normal mode
	tools := &ToolInfo{
		Tools: map[string]int{"Read": 10, "Write": 5, "Bash": 3},
	}
	cfg := Config{
		Features: FeatureToggles{Tools: true},
	}

	lines := renderNormalMode(RenderContext{Data: d, Metrics: m, Tools: tools, Config: cfg})

	// Should have at least 3 lines (line1 + line2 empty or not + line3 with tools)
	if len(lines) < 2 {
		t.Fatalf("renderNormalMode with tools should have 2+ lines, got %d", len(lines))
	}

	// Join all lines to check for tool names
	output := strings.Join(lines, " ")
	if !strings.Contains(output, "Read") {
		t.Errorf("renderNormalMode with tools missing 'Read' in output: %q", output)
	}
	if !strings.Contains(output, "(10)") {
		t.Errorf("renderNormalMode with tools missing count '(10)' in output: %q", output)
	}
}

func TestRenderNormalMode_WithAgents(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
	}
	m := Metrics{ContextPercent: 50}
	tools := &ToolInfo{
		Agents: []string{"researcher", "coder"},
	}
	cfg := Config{
		Features: FeatureToggles{Agents: true},
	}

	lines := renderNormalMode(RenderContext{Data: d, Metrics: m, Tools: tools, Config: cfg})

	// Should have at least 2 lines (line1 + line3 with agents)
	if len(lines) < 2 {
		t.Fatalf("renderNormalMode with agents should have 2+ lines, got %d", len(lines))
	}

	output := strings.Join(lines, " ")
	if !strings.Contains(output, "researcher") {
		t.Errorf("renderNormalMode with agents missing 'researcher' in output: %q", output)
	}
	if !strings.Contains(output, "coder") {
		t.Errorf("renderNormalMode with agents missing 'coder' in output: %q", output)
	}
}

func TestRenderNormalMode_Line4Metrics(t *testing.T) {
	t.Parallel()

	cache := 75
	apiRatio := 40
	costPerMin := 0.15

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
	}
	m := Metrics{
		ContextPercent:  50,
		CacheEfficiency: &cache,
		APIWaitRatio:    &apiRatio,
		CostPerMinute:   &costPerMin,
	}
	cfg := Config{
		Features: FeatureToggles{
			CacheEfficiency: true,
			APIWaitRatio:    true,
			CostVelocity:    true,
		},
	}

	lines := renderNormalMode(RenderContext{Data: d, Metrics: m, Config: cfg})

	// Should have at least 2 lines (line1 + line4)
	if len(lines) < 2 {
		t.Fatalf("renderNormalMode with line4 metrics should have 2+ lines, got %d", len(lines))
	}

	output := strings.Join(lines, " ")
	// Check for cache label
	if !strings.Contains(output, "Cache:") {
		t.Errorf("renderNormalMode line4 missing 'Cache:' in output: %q", output)
	}
	// Check for wait label
	if !strings.Contains(output, "Wait:") {
		t.Errorf("renderNormalMode line4 missing 'Wait:' in output: %q", output)
	}
	// Check for cost label
	if !strings.Contains(output, "Cost:") {
		t.Errorf("renderNormalMode line4 missing 'Cost:' in output: %q", output)
	}
}

func TestRenderNormalMode_VimAndAgent(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
		Vim:           &Vim{Mode: "normal"},
		Agent:         &Agent{Name: "my-agent"},
	}
	m := Metrics{ContextPercent: 50}
	cfg := Config{
		Features: FeatureToggles{
			VimMode:   true,
			AgentName: true,
		},
	}

	lines := renderNormalMode(RenderContext{Data: d, Metrics: m, Config: cfg})

	// Should have at least 2 lines (line1 + line4)
	if len(lines) < 2 {
		t.Fatalf("renderNormalMode with vim/agent should have 2+ lines, got %d", len(lines))
	}

	output := strings.Join(lines, " ")
	// Vim normal mode shows "N"
	if !strings.Contains(output, "N") {
		t.Errorf("renderNormalMode with vim missing 'N' in output: %q", output)
	}
	// Agent name shows @my-agent
	if !strings.Contains(output, "@my-agent") {
		t.Errorf("renderNormalMode with agent missing '@my-agent' in output: %q", output)
	}
}

func TestRenderNormalMode_AllLinesPresent(t *testing.T) {
	t.Parallel()

	cache := 80
	apiRatio := 30
	costPerMin := 0.05
	speed := 100

	d := &StdinData{
		Model:         Model{DisplayName: "Claude Sonnet 4.5"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5, TotalLinesAdded: 50, TotalLinesRemoved: 10},
		Vim:           &Vim{Mode: "insert"},
		Agent:         &Agent{Name: "researcher"},
	}
	m := Metrics{
		ContextPercent:  50,
		CacheEfficiency: &cache,
		APIWaitRatio:    &apiRatio,
		CostPerMinute:   &costPerMin,
		ResponseSpeed:   &speed,
	}
	git := &GitInfo{Branch: "main", Dirty: true}
	tools := &ToolInfo{
		Tools:  map[string]int{"Read": 15, "Write": 8},
		Agents: []string{"coder", "tester"},
	}
	account := &AccountInfo{EmailAddress: "test@example.com"}
	usage := &UsageData{RemainingPercent5h: 60.0}

	cfg := DefaultConfig() // All features enabled

	lines := renderNormalMode(RenderContext{Data: d, Metrics: m, Git: git, Usage: usage, Tools: tools, Account: account, Config: cfg})

	// With all features enabled, should have 4 lines
	if len(lines) != 4 {
		t.Errorf("renderNormalMode with all features should have 4 lines, got %d\nLines: %v", len(lines), lines)
	}

	output := strings.Join(lines, " ")
	// Verify key elements from each line
	if !strings.Contains(output, "Sonnet") {
		t.Errorf("renderNormalMode missing model name in output: %q", output)
	}
	if !strings.Contains(output, "test@example.com") {
		t.Errorf("renderNormalMode missing account in output: %q", output)
	}
	if !strings.Contains(output, "Read") {
		t.Errorf("renderNormalMode missing tools in output: %q", output)
	}
	if !strings.Contains(output, "Cache:") {
		t.Errorf("renderNormalMode missing cache metric in output: %q", output)
	}
}

// renderDangerMode coverage improvements

func TestRenderDangerMode_Full(t *testing.T) {
	t.Parallel()

	cache := 70
	costPerMin := 0.25
	speed := 80

	d := &StdinData{
		Model: Model{DisplayName: "Claude Opus 4"},
		ContextWindow: ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens:          50000,
				OutputTokens:         10000,
				CacheReadInputTokens: 120000,
			},
		},
		Cost:      Cost{TotalDurationMS: 300000, TotalCostUSD: 15.0, TotalLinesAdded: 200, TotalLinesRemoved: 50},
		Workspace: Workspace{ProjectDir: "/home/user/project"},
	}
	m := Metrics{
		ContextPercent:  90,
		CacheEfficiency: &cache,
		CostPerMinute:   &costPerMin,
		ResponseSpeed:   &speed,
	}
	git := &GitInfo{Branch: "feature", Dirty: true}
	usage := &UsageData{RemainingPercent5h: 40.0, RemainingPercent7d: 70.0}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Git: git, Usage: usage, Config: Config{Thresholds: DefaultThresholds()}})

	// Danger mode always returns exactly 2 lines
	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// L1: model | 🔴 context bar (remaining+ETA) | 5h quota bar | 7d quota bar
	line1 := lines[0]
	if !strings.Contains(line1, "Opus 4") {
		t.Errorf("L1 missing model badge: %q", line1)
	}
	if !strings.Contains(line1, "🔴") {
		t.Errorf("L1 missing danger context indicator: %q", line1)
	}
	if !strings.Contains(line1, "90%") {
		t.Errorf("L1 missing context percent: %q", line1)
	}
	if !strings.Contains(line1, "left") {
		t.Errorf("L1 missing remaining tokens: %q", line1)
	}
	if !strings.Contains(line1, "40%") {
		t.Errorf("L1 missing 5h quota: %q", line1)
	}
	if !strings.Contains(line1, "70%") {
		t.Errorf("L1 missing 7d quota: %q", line1)
	}

	// L2: workspace/git | Δchanges | In:XK Out:XK | C:X% | speed | $cost $cost/h | duration
	line2 := lines[1]
	if !strings.Contains(line2, "project/") {
		t.Errorf("L2 missing workspace: %q", line2)
	}
	if !strings.Contains(line2, "feature*") {
		t.Errorf("L2 missing git branch: %q", line2)
	}
	if !strings.Contains(line2, "+200") {
		t.Errorf("L2 missing line changes: %q", line2)
	}
	if !strings.Contains(line2, "In:50K") {
		t.Errorf("L2 missing token IO: %q", line2)
	}
	if !strings.Contains(line2, "C70%") {
		t.Errorf("L2 missing cache efficiency: %q", line2)
	}
	if !strings.Contains(line2, "80tok/s") {
		t.Errorf("L2 missing speed: %q", line2)
	}
	// Cost per hour (0.25 * 60 = 15.0/h)
	if !strings.Contains(line2, "15.0/h") {
		t.Errorf("L2 missing cost/hour: %q", line2)
	}
	if !strings.Contains(line2, "5m") {
		t.Errorf("L2 missing duration: %q", line2)
	}
}

func TestRenderDangerMode_WithWorkspaceAndGit(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
		Workspace:     Workspace{ProjectDir: "/home/user/myproject"},
	}
	m := Metrics{ContextPercent: 85}
	git := &GitInfo{Branch: "main"}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Git: git, Config: Config{Thresholds: DefaultThresholds()}})

	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// Check line2 contains workspace + git combined
	if !strings.Contains(lines[1], "myproject/") {
		t.Errorf("renderDangerMode missing workspace in line2: %q", lines[1])
	}
	if !strings.Contains(lines[1], "main") {
		t.Errorf("renderDangerMode missing git branch in line2: %q", lines[1])
	}
}

func TestRenderDangerMode_WithWorkspaceNoGit(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
		Workspace:     Workspace{CurrentDir: "/home/user/test"},
	}
	m := Metrics{ContextPercent: 90}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Config: Config{Thresholds: DefaultThresholds()}})

	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// Check line2 contains workspace only (no git)
	if !strings.Contains(lines[1], "test/") {
		t.Errorf("renderDangerMode missing workspace in line2: %q", lines[1])
	}
}

func TestRenderDangerMode_WithQuota(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
	}
	m := Metrics{ContextPercent: 92}
	usage := &UsageData{RemainingPercent5h: 15.0, RemainingPercent7d: 80.0}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Usage: usage, Config: Config{Thresholds: DefaultThresholds()}})

	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// Quota bars are now on line1
	if !strings.Contains(lines[0], "15%") {
		t.Errorf("renderDangerMode missing 5h quota in line1: %q", lines[0])
	}
	if !strings.Contains(lines[0], "80%") {
		t.Errorf("renderDangerMode missing 7d quota in line1: %q", lines[0])
	}
}

func TestRenderDangerMode_WithVimAndAgent(t *testing.T) {
	t.Parallel()

	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
	}
	m := Metrics{ContextPercent: 88}

	// In new danger layout, vim/agent are not shown — compact 2-line layout
	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Config: Config{Thresholds: DefaultThresholds()}})

	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// L1 should have danger context bar with ETA
	if !strings.Contains(lines[0], "🔴") {
		t.Errorf("L1 missing danger indicator: %q", lines[0])
	}
	if !strings.Contains(lines[0], "88%") {
		t.Errorf("L1 missing context percent: %q", lines[0])
	}
	if !strings.Contains(lines[0], "left") {
		t.Errorf("L1 missing remaining tokens info: %q", lines[0])
	}

	// L2 should have duration
	if !strings.Contains(lines[1], "1m") {
		t.Errorf("L2 missing duration: %q", lines[1])
	}
}

func TestRenderDangerMode_CostPerHour(t *testing.T) {
	t.Parallel()

	costPerMin := 0.50 // 0.50/min * 60 = 30.0/h

	d := &StdinData{
		Model:         Model{DisplayName: "Opus"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 180000, TotalCostUSD: 10.0},
	}
	m := Metrics{
		ContextPercent: 87,
		CostPerMinute:  &costPerMin,
	}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Config: Config{Thresholds: DefaultThresholds()}})

	if len(lines) != 2 {
		t.Fatalf("renderDangerMode should return 2 lines, got %d", len(lines))
	}

	// Check for cost per hour format: $30.0/h
	if !strings.Contains(lines[1], "30.0/h") {
		t.Errorf("renderDangerMode missing cost/hour '30.0/h' in line2: %q", lines[1])
	}
}

func TestRenderDangerMode_MinimalData(t *testing.T) {
	t.Parallel()

	// Only required fields: model, context window, duration
	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 60000},
	}
	m := Metrics{ContextPercent: 85}

	lines := renderDangerMode(RenderContext{Data: d, Metrics: m, Config: Config{Thresholds: DefaultThresholds()}})

	// Should not panic and should return 2 lines
	if len(lines) != 2 {
		t.Fatalf("renderDangerMode with minimal data should return 2 lines, got %d", len(lines))
	}

	// Line1 should contain model and context bar
	if !strings.Contains(lines[0], "Sonnet") {
		t.Errorf("renderDangerMode line1 missing model name: %q", lines[0])
	}
	if !strings.Contains(lines[0], "85%") {
		t.Errorf("renderDangerMode line1 missing context percentage: %q", lines[0])
	}
}

// PresetConfig edge case

func TestPresetConfig_UnknownPreset(t *testing.T) {
	t.Parallel()

	cfg := PresetConfig("garbage")

	// Unknown preset should fallback to "full"
	if cfg.Preset != "full" {
		t.Errorf("PresetConfig with unknown preset should return 'full', got %q", cfg.Preset)
	}

	// Should have all features enabled (full preset)
	if !cfg.Features.Account {
		t.Error("PresetConfig fallback to full should have Account enabled")
	}
	if !cfg.Features.Git {
		t.Error("PresetConfig fallback to full should have Git enabled")
	}
	if !cfg.Features.Tools {
		t.Error("PresetConfig fallback to full should have Tools enabled")
	}
}

// ====== Custom Thresholds Tests ======

func TestContextColor_CustomThresholds(t *testing.T) {
	t.Parallel()

	customThresholds := DefaultThresholds()
	customThresholds.ContextDanger = 90
	customThresholds.ContextWarning = 75
	customThresholds.ContextModerate = 40

	tests := []struct {
		name    string
		percent int
		want    string
	}{
		{"85 is orange not boldRed (danger=90)", 85, orange},
		{"90 is boldRed (at danger)", 90, boldRed},
		{"74 is yellow (below warning=75)", 74, yellow},
		{"75 is orange (at warning)", 75, orange},
		{"91 is boldRed (above danger)", 91, boldRed},
		{"40 is yellow (at custom moderate=40)", 40, yellow},
		{"39 is green (below custom moderate=40)", 39, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contextColor(tt.percent, customThresholds)
			if got != tt.want {
				t.Errorf("contextColor(%d, custom) = %q, want %q", tt.percent, got, tt.want)
			}
		})
	}
}

func TestRenderResponseSpeed_CustomThresholds(t *testing.T) {
	t.Parallel()

	customThresholds := DefaultThresholds()
	customThresholds.SpeedFast = 100
	customThresholds.SpeedModerate = 50

	tests := []struct {
		name      string
		speed     int
		wantColor string
	}{
		{"60 is yellow (below fast=100, above moderate=50)", 60, yellow},
		{"100 is green (at fast)", 100, green},
		{"49 is orange (below moderate=50)", 49, orange},
		{"101 is green (above fast)", 101, green},
		{"50 is yellow (at moderate)", 50, yellow},
		{"20 is orange (below moderate)", 20, orange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderResponseSpeed(tt.speed, customThresholds)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderResponseSpeed(%d, custom) missing color %q in %q", tt.speed, tt.wantColor, got)
			}
			if !strings.Contains(got, "tok/s") {
				t.Errorf("renderResponseSpeed(%d, custom) missing 'tok/s' in %q", tt.speed, got)
			}
		})
	}
}

func TestRender_CustomDangerThreshold(t *testing.T) {
	t.Parallel()

	cache := 80
	apiRatio := 35
	speed := 60
	d := &StdinData{
		Model:         Model{DisplayName: "Sonnet"},
		ContextWindow: ContextWindow{ContextWindowSize: 200000},
		Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.0, TotalLinesAdded: 50, TotalLinesRemoved: 10},
		Workspace:     Workspace{ProjectDir: "/test"},
		Vim:           &Vim{Mode: "normal"},
	}
	git := &GitInfo{Branch: "main", Dirty: true}
	account := &AccountInfo{EmailAddress: "test@example.com"}
	usage := &UsageData{RemainingPercent5h: 60.0, RemainingPercent7d: 80.0}
	cfg := DefaultConfig()
	cfg.Thresholds.ContextDanger = 90

	t.Run("87% with danger=90 should be normal mode", func(t *testing.T) {
		m := Metrics{
			ContextPercent:  87,
			CacheEfficiency: &cache,
			APIWaitRatio:    &apiRatio,
			ResponseSpeed:   &speed,
		}
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Usage: usage, Account: account, Config: cfg})

		// Normal mode with usage should have more than 2 lines (danger mode always returns exactly 2)
		if len(lines) <= 2 {
			t.Errorf("Render with 87%% and danger=90 should be normal mode (>2 lines), got %d lines", len(lines))
		}

		// Should NOT contain danger indicator
		output := strings.Join(lines, " ")
		if strings.Contains(output, "🔴") {
			t.Error("Normal mode should not contain danger indicator 🔴")
		}
	})

	t.Run("90% with danger=90 should be danger mode", func(t *testing.T) {
		m := Metrics{ContextPercent: 90}
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Config: cfg})

		// Danger mode should return exactly 2 lines
		if len(lines) != 2 {
			t.Errorf("Render with 90%% and danger=90 should be danger mode (2 lines), got %d lines", len(lines))
		}

		// Should contain danger indicator
		output := strings.Join(lines, " ")
		if strings.Contains(output, "🔴") == false {
			t.Error("Danger mode should contain danger indicator 🔴")
		}
	})

	t.Run("91% with danger=90 should be danger mode", func(t *testing.T) {
		m := Metrics{ContextPercent: 91}
		lines := Render(RenderContext{Data: d, Metrics: m, Git: git, Config: cfg})

		if len(lines) != 2 {
			t.Errorf("Render with 91%% and danger=90 should be danger mode (2 lines), got %d lines", len(lines))
		}
	})
}

// ====== Custom Threshold Tests (Task #2) ======

func TestRenderCost_CustomThresholds(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.SessionCostHigh = 10.0
	custom.SessionCostMedium = 3.0

	tests := []struct {
		name      string
		usd       float64
		wantColor string
	}{
		{"$4.99 is yellow (between 3.0 and 10.0)", 4.99, yellow},
		{"$2.99 is default (below 3.0)", 2.99, Reset},
		{"$10.0 is boldRed (at high)", 10.0, boldRed},
		{"$15.0 is boldRed (above high)", 15.0, boldRed},
		{"$3.0 is yellow (at medium)", 3.0, yellow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderCost(tt.usd, custom)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCost(%f, custom) missing color %q in %q", tt.usd, tt.wantColor, got)
			}
		})
	}
}

func TestCacheColor_CustomThresholds(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.CacheExcellent = 90
	custom.CacheGood = 60

	tests := []struct {
		name      string
		pct       int
		wantColor string
	}{
		{"75% is yellow (between 60-90)", 75, yellow},
		{"59% is red (below 60)", 59, red},
		{"90% is green (at excellent)", 90, green},
		{"60% is yellow (at good)", 60, yellow},
		{"95% is green (above excellent)", 95, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cacheColor(tt.pct, custom)
			if got != tt.wantColor {
				t.Errorf("cacheColor(%d, custom) = %q, want %q", tt.pct, got, tt.wantColor)
			}
		})
	}
}

func TestApiRatioColor_CustomThresholds(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.WaitHigh = 80
	custom.WaitMedium = 50

	tests := []struct {
		name      string
		pct       int
		wantColor string
	}{
		{"45% is green (below 50)", 45, green},
		{"55% is yellow (between 50-80)", 55, yellow},
		{"80% is red (at high)", 80, red},
		{"50% is yellow (at medium)", 50, yellow},
		{"90% is red (above high)", 90, red},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := apiRatioColor(tt.pct, custom)
			if got != tt.wantColor {
				t.Errorf("apiRatioColor(%d, custom) = %q, want %q", tt.pct, got, tt.wantColor)
			}
		})
	}
}

func TestQuotaColor_CustomThresholds(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.QuotaCritical = 20
	custom.QuotaLow = 40
	custom.QuotaMedium = 60
	custom.QuotaHigh = 80

	tests := []struct {
		name      string
		remaining float64
		wantColor string
	}{
		{"15% is boldRed (below critical=20)", 15, boldRed},
		{"30% is red (between critical=20 and low=40)", 30, red},
		{"50% is orange (between low=40 and medium=60)", 50, orange},
		{"70% is yellow (between medium=60 and high=80)", 70, yellow},
		{"85% is green (above high=80)", 85, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := quotaColor(tt.remaining, custom)
			if got != tt.wantColor {
				t.Errorf("quotaColor(%f, custom) = %q, want %q", tt.remaining, got, tt.wantColor)
			}
		})
	}
}

func TestRenderCostVelocityLabeled_CustomThresholds(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.CostVelocityHigh = 1.0
	custom.CostVelocityMedium = 0.30

	tests := []struct {
		name      string
		perMin    float64
		wantColor string
	}{
		{"$0.20/min is green (below 0.30)", 0.20, green},
		{"$0.50/min is yellow (between 0.30-1.0)", 0.50, yellow},
		{"$1.0/min is boldRed (at high)", 1.0, boldRed},
		{"$0.30/min is yellow (at medium)", 0.30, yellow},
		{"$2.0/min is boldRed (above high)", 2.0, boldRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderCostVelocityLabeled(tt.perMin, custom)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCostVelocityLabeled(%f, custom) missing color %q in %q", tt.perMin, tt.wantColor, got)
			}
			if !strings.Contains(got, "Cost:") {
				t.Errorf("renderCostVelocityLabeled(%f, custom) missing 'Cost:' in %q", tt.perMin, got)
			}
		})
	}
}

func TestRenderContextBar_CustomThresholdIcons(t *testing.T) {
	t.Parallel()

	custom := DefaultThresholds()
	custom.ContextDanger = 90
	custom.ContextWarning = 75
	custom.ContextModerate = 40

	cw := ContextWindow{ContextWindowSize: 200000}

	tests := []struct {
		name       string
		percent    int
		wantIcon   string
		wantNoIcon string
	}{
		{"80% has warning icon", 80, "⚠", "🔴"},
		{"85% has warning icon (not danger, danger=90)", 85, "⚠", "🔴"},
		{"90% has danger icon", 90, "🔴", ""},
		{"74% has no icon (below warning=75)", 74, "", "⚠"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderContextBar(tt.percent, cw, custom)
			if tt.wantIcon != "" && !strings.Contains(got, tt.wantIcon) {
				t.Errorf("renderContextBar(%d, custom) missing icon %q in %q", tt.percent, tt.wantIcon, got)
			}
			if tt.wantNoIcon != "" && strings.Contains(got, tt.wantNoIcon) {
				t.Errorf("renderContextBar(%d, custom) should NOT contain icon %q in %q", tt.percent, tt.wantNoIcon, got)
			}
		})
	}
}
