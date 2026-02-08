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
			got := renderModelBadge(tt.model)
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
			got := contextColor(tt.percent)
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
		{"small cost white", 0.50, false, white, "$0.50"},
		{"medium cost yellow", 1.50, false, yellow, "$1.50"},
		{"high cost boldRed", 5.50, false, boldRed, "$5.50"},
		{">=10 uses one decimal", 15.0, false, boldRed, "$15.0"},
		{"exactly 1.0 is yellow", 1.0, false, yellow, "$1.00"},
		{"exactly 5.0 is boldRed", 5.0, false, boldRed, "$5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCost(tt.usd)
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
			name:        "empty email",
			acc:         &AccountInfo{EmailAddress: ""},
			wantContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAccount(tt.acc)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("renderAccount() = %q, want to contain %q", got, tt.wantContain)
			}
			// Verify grey color is used (if email not empty)
			if tt.wantContain != "" && !strings.Contains(got, grey) {
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
		{"normal", "normal", blue + "N" + Reset},
		{"NORMAL uppercase", "NORMAL", blue + "N" + Reset},
		{"insert", "insert", green + "I" + Reset},
		{"visual", "visual", magenta + "V" + Reset},
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
			got := renderCacheEfficiencyCompact(tt.pct)
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
		wantColor string
	}{
		{"excellent green", 80, green},
		{"good yellow", 50, yellow},
		{"poor red", 49, red},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCacheEfficiencyLabeled(tt.pct)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCacheEfficiencyLabeled(%d) missing color in %q", tt.pct, got)
			}
			if !strings.Contains(got, "Cache:") {
				t.Errorf("renderCacheEfficiencyLabeled(%d) missing 'Cache:' label in %q", tt.pct, got)
			}
		})
	}
}

func TestRenderAPIRatioCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pct       int
		wantColor string
	}{
		{"high >=60 red", 60, red},
		{"medium >=35 yellow", 35, yellow},
		{"low <35 green", 34, green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAPIRatioCompact(tt.pct)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderAPIRatioCompact(%d) missing color %q in %q", tt.pct, tt.wantColor, got)
			}
			if !strings.Contains(got, "A") {
				t.Errorf("renderAPIRatioCompact(%d) missing 'A' prefix in %q", tt.pct, got)
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
			got := renderAPIRatioLabeled(tt.pct)
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
			got := renderResponseSpeed(tt.speed)
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
			got := renderCostVelocityLabeled(tt.perMin)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("renderCostVelocityLabeled(%f) missing color in %q", tt.perMin, got)
			}
			if !strings.Contains(got, "Cost:") {
				t.Errorf("renderCostVelocityLabeled(%f) missing 'Cost:' in %q", tt.perMin, got)
			}
		})
	}
}

func TestRenderTokenBreakdown(t *testing.T) {
	t.Parallel()

	cu := &CurrentUsage{
		InputTokens:          30000,
		OutputTokens:         3000,
		CacheReadInputTokens: 135000,
	}
	got := renderTokenBreakdown(cu)
	if !strings.Contains(got, "In:30K") {
		t.Errorf("renderTokenBreakdown() missing In:30K in %q", got)
	}
	if !strings.Contains(got, "Out:3K") {
		t.Errorf("renderTokenBreakdown() missing Out:3K in %q", got)
	}
	if !strings.Contains(got, "Cache:135K") {
		t.Errorf("renderTokenBreakdown() missing Cache:135K in %q", got)
	}
}

func TestRenderTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tools     map[string]int
		wantEmpty bool
		wantParts []string
	}{
		{"nil map", nil, true, nil},
		{"empty map", map[string]int{}, true, nil},
		{"single tool", map[string]int{"Read": 5}, false, []string{"Read", "(5)"}},
		{
			"more than 5 tools keeps top 5",
			map[string]int{"A": 1, "B": 2, "C": 3, "D": 4, "E": 5, "F": 6, "G": 7},
			false,
			[]string{"G", "(7)", "F", "(6)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTools(tt.tools)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("renderTools(%v) = %q, want empty", tt.tools, got)
				}
				return
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("renderTools(%v) missing %q in %q", tt.tools, part, got)
				}
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
			Cost:          Cost{TotalDurationMS: 120000, TotalCostUSD: 1.5},
		}
		m := Metrics{ContextPercent: 50}
		lines := Render(d, m, nil, nil, nil, nil)
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
		lines := Render(d, m, nil, nil, nil, nil)
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
		lines := Render(d, m, nil, nil, nil, nil)
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
		lines := Render(d, m, git, nil, nil, nil)
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
			got := renderContextBar(tt.percent, cw)
			for _, want := range tt.wantIn {
				if !strings.Contains(got, want) {
					t.Errorf("renderContextBar(%d, %d) missing %q in %q", tt.percent, tt.cwSize, want, got)
				}
			}
		})
	}
}
