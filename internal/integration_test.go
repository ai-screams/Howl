package internal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// JSON Fixtures: Real-world JSON strings for full pipeline testing
const (
	// Normal session: 42% context, Opus 4.6, $1.50 cost, 120s duration, 200K context window
	jsonNormalSession = `{
  "session_id": "session-1",
  "model": {"id": "claude-opus-4.6", "display_name": "Opus 4.6"},
  "context_window": {
    "total_input_tokens": 42000,
    "total_output_tokens": 8000,
    "context_window_size": 200000,
    "used_percentage": 42.0,
    "current_usage": {
      "input_tokens": 5000,
      "output_tokens": 1000,
      "cache_creation_input_tokens": 2000,
      "cache_read_input_tokens": 3000
    }
  },
  "cost": {
    "total_cost_usd": 1.50,
    "total_duration_ms": 120000,
    "total_api_duration_ms": 60000,
    "total_lines_added": 250,
    "total_lines_removed": 100
  },
  "workspace": {"project_dir": "/Users/test/my-project"}
}`

	// Danger session: 90% context, Sonnet 4.5, $15.00 cost, 300s duration
	jsonDangerSession = `{
  "session_id": "session-2",
  "model": {"id": "claude-sonnet-4.5", "display_name": "Sonnet 4.5"},
  "context_window": {
    "total_input_tokens": 180000,
    "total_output_tokens": 20000,
    "context_window_size": 200000,
    "used_percentage": 90.0,
    "current_usage": {
      "input_tokens": 10000,
      "output_tokens": 2000,
      "cache_creation_input_tokens": 1000,
      "cache_read_input_tokens": 8000
    }
  },
  "cost": {
    "total_cost_usd": 15.00,
    "total_duration_ms": 300000,
    "total_api_duration_ms": 200000,
    "total_lines_added": 1500,
    "total_lines_removed": 500
  },
  "workspace": {"project_dir": "/Users/test/danger-project"}
}`

	// Minimal session: only model and context window size
	jsonMinimalSession = `{
  "model": {"id": "claude-haiku", "display_name": "Haiku"},
  "context_window": {"context_window_size": 200000},
  "cost": {},
  "workspace": {}
}`

	// Boundary 85%: exactly at danger threshold
	jsonBoundary85 = `{
  "model": {"id": "claude-opus-4.6"},
  "context_window": {
    "total_input_tokens": 170000,
    "total_output_tokens": 10000,
    "context_window_size": 200000,
    "used_percentage": 85.0,
    "current_usage": {
      "input_tokens": 10000,
      "output_tokens": 1000,
      "cache_creation_input_tokens": 5000,
      "cache_read_input_tokens": 5000
    }
  },
  "cost": {
    "total_cost_usd": 5.00,
    "total_duration_ms": 180000,
    "total_api_duration_ms": 100000,
    "total_lines_added": 800,
    "total_lines_removed": 200
  },
  "workspace": {"project_dir": "/Users/test/edge-project"}
}`

	// Boundary 84%: just below danger threshold
	jsonBoundary84 = `{
  "model": {"id": "claude-sonnet-4.5"},
  "context_window": {
    "total_input_tokens": 168000,
    "total_output_tokens": 8000,
    "context_window_size": 200000,
    "used_percentage": 84.0,
    "current_usage": {
      "input_tokens": 8000,
      "output_tokens": 1000,
      "cache_creation_input_tokens": 4000,
      "cache_read_input_tokens": 4000
    }
  },
  "cost": {
    "total_cost_usd": 4.50,
    "total_duration_ms": 150000,
    "total_api_duration_ms": 80000,
    "total_lines_added": 600,
    "total_lines_removed": 150
  },
  "workspace": {"project_dir": "/Users/test/edge-project"}
}`

	// Full exhausted: 100% context used
	jsonFullExhausted = `{
  "model": {"id": "claude-opus-4.6"},
  "context_window": {
    "total_input_tokens": 200000,
    "total_output_tokens": 0,
    "context_window_size": 200000,
    "used_percentage": 100.0,
    "current_usage": {
      "input_tokens": 10000,
      "output_tokens": 0,
      "cache_creation_input_tokens": 5000,
      "cache_read_input_tokens": 5000
    }
  },
  "cost": {
    "total_cost_usd": 20.00,
    "total_duration_ms": 400000,
    "total_api_duration_ms": 300000,
    "total_lines_added": 2000,
    "total_lines_removed": 800
  },
  "workspace": {"project_dir": "/Users/test/maxed"}
}`

	// Fresh session: 0% context, 0 cost, 0 duration
	jsonFreshSession = `{
  "model": {"id": "claude-haiku"},
  "context_window": {
    "total_input_tokens": 0,
    "total_output_tokens": 0,
    "context_window_size": 200000,
    "used_percentage": 0.0
  },
  "cost": {
    "total_cost_usd": 0.0,
    "total_duration_ms": 0,
    "total_api_duration_ms": 0,
    "total_lines_added": 0,
    "total_lines_removed": 0
  },
  "workspace": {}
}`

	// Vim and Agent: includes vim mode and agent name
	jsonVimAndAgent = `{
  "model": {"id": "claude-sonnet-4.5"},
  "context_window": {
    "total_input_tokens": 50000,
    "total_output_tokens": 5000,
    "context_window_size": 200000,
    "used_percentage": 27.5,
    "current_usage": {
      "input_tokens": 5000,
      "output_tokens": 500,
      "cache_creation_input_tokens": 2000,
      "cache_read_input_tokens": 3000
    }
  },
  "cost": {
    "total_cost_usd": 2.00,
    "total_duration_ms": 90000,
    "total_api_duration_ms": 45000,
    "total_lines_added": 300,
    "total_lines_removed": 50
  },
  "vim": {"mode": "normal"},
  "agent": {"name": "my-agent"},
  "workspace": {}
}`

	// Bedrock model: model ID contains "anthropic.claude-"
	jsonBedrockModel = `{
  "model": {"id": "anthropic.claude-opus-4.0", "display_name": "Opus 4.0"},
  "context_window": {
    "total_input_tokens": 30000,
    "total_output_tokens": 3000,
    "context_window_size": 200000,
    "used_percentage": 16.5
  },
  "cost": {
    "total_cost_usd": 1.20,
    "total_duration_ms": 60000,
    "total_api_duration_ms": 30000
  },
  "workspace": {}
}`

	// 1M context window
	json1MContext = `{
  "model": {"id": "claude-opus-4.6"},
  "context_window": {
    "total_input_tokens": 500000,
    "total_output_tokens": 50000,
    "context_window_size": 1000000,
    "used_percentage": 55.0,
    "current_usage": {
      "input_tokens": 50000,
      "output_tokens": 5000,
      "cache_creation_input_tokens": 20000,
      "cache_read_input_tokens": 30000
    }
  },
  "cost": {
    "total_cost_usd": 10.00,
    "total_duration_ms": 200000,
    "total_api_duration_ms": 120000,
    "total_lines_added": 1000,
    "total_lines_removed": 200
  },
  "workspace": {}
}`

	// Empty JSON object
	jsonZeroStruct = `{}`
)

func TestIntegration_NormalMode(t *testing.T) {
	t.Parallel()

	futureTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name             string
		json             string
		cfg              Config
		git              *GitInfo
		usage            *UsageData
		tools            *ToolInfo
		account          *AccountInfo
		wantMinLines     int
		wantMaxLines     int
		wantContains     []string
		wantNotContain   []string
		wantLineContains map[int][]string // line index -> required substrings
	}{
		{
			name:         "normal_session_full_preset",
			json:         jsonNormalSession,
			cfg:          PresetConfig("full"),
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"Opus",
				"$1.50",
				"42%",
				"+250",
				"-100",
			},
			wantNotContain: []string{"🔴", "danger"},
		},
		{
			name:         "minimal_session",
			json:         jsonMinimalSession,
			cfg:          PresetConfig("minimal"),
			wantMinLines: 1,
			wantMaxLines: 4,
			wantContains: []string{"Haiku"},
			wantNotContain: []string{
				"$", // no cost
			},
		},
		{
			name:         "boundary_84_normal_mode",
			json:         jsonBoundary84,
			cfg:          PresetConfig("full"),
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"claude-sonnet-4.5", // model ID
				"84%",
				"⚠", // warning icon (70%+)
			},
			wantNotContain: []string{"🔴"}, // not danger mode
		},
		{
			name:         "fresh_session",
			json:         jsonFreshSession,
			cfg:          PresetConfig("full"),
			wantMinLines: 1,
			wantMaxLines: 4,
			wantContains: []string{
				"claude-haiku", // model ID (ANSI codes present)
				"0%",
				"<1m",
			},
			wantNotContain: []string{"$"}, // no cost displayed
		},
		{
			name:         "vim_and_agent_full_config",
			json:         jsonVimAndAgent,
			cfg:          PresetConfig("full"),
			wantMinLines: 2,
			wantMaxLines: 4,
			wantLineContains: map[int][]string{
				2: {"N", "@my-agent"}, // Line 3 (index 2) has vim and agent
			},
			wantContains: []string{"claude-sonnet-4.5", "27%"},
		},
		{
			name:         "bedrock_model_suffix",
			json:         jsonBedrockModel,
			cfg:          PresetConfig("full"),
			wantMinLines: 1,
			wantMaxLines: 4,
			wantContains: []string{
				"BR", // Bedrock suffix
				"Opus",
			},
		},
		{
			name:         "1M_context_window",
			json:         json1MContext,
			cfg:          PresetConfig("full"),
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"1M", // 1 million tokens formatted as "1M"
				"55%",
			},
		},
		{
			name: "with_usage_quota",
			json: jsonNormalSession,
			cfg:  PresetConfig("full"),
			usage: &UsageData{
				RemainingPercent5h: 80.0,
				RemainingPercent7d: 90.0,
				ResetsAt5h:         futureTime,
				ResetsAt7d:         futureTime.Add(48 * time.Hour),
				FetchedAt:          time.Now().Unix(),
			},
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"80%", // 5h quota
				"90%", // 7d quota
			},
		},
		{
			name:         "developer_preset_2_lines",
			json:         jsonNormalSession,
			cfg:          PresetConfig("developer"),
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"Opus",
				"+250",
			},
		},
		{
			name: "with_git_info",
			json: jsonNormalSession,
			cfg:  PresetConfig("full"),
			git: &GitInfo{
				Branch: "main",
				Dirty:  true,
			},
			wantMinLines: 2,
			wantMaxLines: 4,
			wantContains: []string{
				"main*", // dirty indicator
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d StdinData
			if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
				t.Fatalf("JSON parse error: %v", err)
			}

			m := ComputeMetrics(&d)
			lines := Render(&d, m, tt.git, tt.usage, tt.tools, tt.account, tt.cfg)

			// Verify line count
			if len(lines) < tt.wantMinLines {
				t.Errorf("got %d lines, want at least %d", len(lines), tt.wantMinLines)
			}
			if len(lines) > tt.wantMaxLines {
				t.Errorf("got %d lines, want at most %d", len(lines), tt.wantMaxLines)
			}

			output := strings.Join(lines, "\n")

			// Verify contains
			for _, want := range tt.wantContains {
				if want != "" && !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}

			// Verify not contains
			for _, unwanted := range tt.wantNotContain {
				if unwanted != "" && strings.Contains(output, unwanted) {
					t.Errorf("output contains unwanted %q\nGot:\n%s", unwanted, output)
				}
			}

			// Verify specific line contains
			for lineIdx, wants := range tt.wantLineContains {
				if lineIdx >= len(lines) {
					t.Errorf("line %d does not exist (only %d lines)", lineIdx, len(lines))
					continue
				}
				for _, want := range wants {
					if want != "" && !strings.Contains(lines[lineIdx], want) {
						t.Errorf("line %d missing %q\nLine: %s", lineIdx, want, lines[lineIdx])
					}
				}
			}
		})
	}
}

func TestIntegration_DangerMode(t *testing.T) {
	t.Parallel()

	futureTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name         string
		json         string
		git          *GitInfo
		usage        *UsageData
		tools        *ToolInfo
		wantContains []string
	}{
		{
			name: "danger_session_90_percent",
			json: jsonDangerSession,
			wantContains: []string{
				"🔴",
				"90%",
				"Sonnet",
				"+1.5K", // formatted line changes
				"-500",
			},
		},
		{
			name: "boundary_85_danger_mode",
			json: jsonBoundary85,
			wantContains: []string{
				"🔴",
				"85%",
				"claude-opus-4.6", // model ID
			},
		},
		{
			name: "full_exhausted_100_percent",
			json: jsonFullExhausted,
			wantContains: []string{
				"🔴",
				"100%",
				"+2.0K",
				"-800",
			},
		},
		{
			name: "danger_with_all_optional_data",
			json: jsonDangerSession,
			git: &GitInfo{
				Branch: "feature",
				Dirty:  false,
			},
			usage: &UsageData{
				RemainingPercent5h: 25.0,
				RemainingPercent7d: 40.0,
				ResetsAt5h:         futureTime,
				ResetsAt7d:         futureTime.Add(72 * time.Hour),
				FetchedAt:          time.Now().Unix(),
			},
			wantContains: []string{
				"🔴",
				"90%",
				"feature", // git branch
				"25%",     // 5h quota
				"40%",     // 7d quota
			},
		},
		{
			name: "danger_with_tools",
			json: jsonDangerSession,
			tools: &ToolInfo{
				Tools: map[string]int{
					"Read":  10,
					"Write": 5,
				},
				Agents: []string{},
			},
			wantContains: []string{
				"🔴",
				"90%",
				// Tools not shown in danger mode line 2 (compact mode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d StdinData
			if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
				t.Fatalf("JSON parse error: %v", err)
			}

			m := ComputeMetrics(&d)
			lines := Render(&d, m, tt.git, tt.usage, tt.tools, nil, DefaultConfig())

			// Danger mode always renders exactly 2 lines
			if len(lines) != 2 {
				t.Errorf("danger mode: got %d lines, want exactly 2", len(lines))
			}

			output := strings.Join(lines, "\n")

			for _, want := range tt.wantContains {
				if want != "" && !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestIntegration_JSONDecodingEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		json         string
		wantMinLines int
	}{
		{
			name: "extra_unknown_fields",
			json: `{
  "model": {"id": "claude-haiku"},
  "context_window": {"context_window_size": 200000},
  "cost": {},
  "workspace": {},
  "unknown_field": "ignored",
  "another_unknown": 12345
}`,
			wantMinLines: 1,
		},
		{
			name: "null_optional_fields",
			json: `{
  "model": {"id": "claude-haiku"},
  "context_window": {
    "context_window_size": 200000,
    "current_usage": null,
    "used_percentage": null
  },
  "cost": {},
  "workspace": {},
  "vim": null,
  "agent": null
}`,
			wantMinLines: 1,
		},
		{
			name: "integer_cost_not_float",
			json: `{
  "model": {"id": "claude-haiku"},
  "context_window": {"context_window_size": 200000},
  "cost": {"total_cost_usd": 1},
  "workspace": {}
}`,
			wantMinLines: 1,
		},
		{
			name:         "empty_json_object",
			json:         jsonZeroStruct,
			wantMinLines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d StdinData
			if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
				t.Fatalf("JSON parse error: %v", err)
			}

			m := ComputeMetrics(&d)

			// Should not panic
			lines := Render(&d, m, nil, nil, nil, nil, DefaultConfig())

			if len(lines) < tt.wantMinLines {
				t.Errorf("got %d lines, want at least %d", len(lines), tt.wantMinLines)
			}
		})
	}
}

func TestIntegration_MetricsComputedFromJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		json                    string
		wantContextPercent      int
		wantCacheEfficiencyMin  *int
		wantResponseSpeedMin    *int
		wantCostPerMinuteExists bool
	}{
		{
			name:               "normal_session_metrics",
			json:               jsonNormalSession,
			wantContextPercent: 42,
			wantCacheEfficiencyMin: func() *int {
				v := 30 // At least 30% cache efficiency
				return &v
			}(),
			wantResponseSpeedMin: func() *int {
				v := 120 // 8000 output tokens / 60s = 133 tok/s
				return &v
			}(),
			wantCostPerMinuteExists: true,
		},
		{
			name:               "danger_session_metrics",
			json:               jsonDangerSession,
			wantContextPercent: 90,
			wantCacheEfficiencyMin: func() *int {
				v := 42 // 8000 cache / (10000 + 1000 + 8000) = 42%
				return &v
			}(),
			wantResponseSpeedMin: func() *int {
				v := 90 // 20000 output tokens / 200s = 100 tok/s
				return &v
			}(),
			wantCostPerMinuteExists: true,
		},
		{
			name:                    "fresh_session_no_metrics",
			json:                    jsonFreshSession,
			wantContextPercent:      0,
			wantCacheEfficiencyMin:  nil,
			wantResponseSpeedMin:    nil,
			wantCostPerMinuteExists: false,
		},
		{
			name:               "boundary_85_metrics",
			json:               jsonBoundary85,
			wantContextPercent: 85,
			wantCacheEfficiencyMin: func() *int {
				v := 25 // 5000 cache / (10000 + 5000 + 5000) = 25%
				return &v
			}(),
			wantResponseSpeedMin: func() *int {
				v := 9 // 1000 output / 100s = 10 tok/s
				return &v
			}(),
			wantCostPerMinuteExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d StdinData
			if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
				t.Fatalf("JSON parse error: %v", err)
			}

			m := ComputeMetrics(&d)

			// Verify context percent
			if m.ContextPercent != tt.wantContextPercent {
				t.Errorf("ContextPercent: got %d, want %d", m.ContextPercent, tt.wantContextPercent)
			}

			// Verify cache efficiency
			if tt.wantCacheEfficiencyMin != nil {
				if m.CacheEfficiency == nil {
					t.Errorf("CacheEfficiency: got nil, want >= %d", *tt.wantCacheEfficiencyMin)
				} else if *m.CacheEfficiency < *tt.wantCacheEfficiencyMin {
					t.Errorf("CacheEfficiency: got %d, want >= %d", *m.CacheEfficiency, *tt.wantCacheEfficiencyMin)
				}
			} else {
				if m.CacheEfficiency != nil {
					t.Errorf("CacheEfficiency: got %d, want nil", *m.CacheEfficiency)
				}
			}

			// Verify response speed
			if tt.wantResponseSpeedMin != nil {
				if m.ResponseSpeed == nil {
					t.Errorf("ResponseSpeed: got nil, want >= %d", *tt.wantResponseSpeedMin)
				} else if *m.ResponseSpeed < *tt.wantResponseSpeedMin {
					t.Errorf("ResponseSpeed: got %d, want >= %d", *m.ResponseSpeed, *tt.wantResponseSpeedMin)
				}
			} else {
				if m.ResponseSpeed != nil {
					t.Errorf("ResponseSpeed: got %d, want nil", *m.ResponseSpeed)
				}
			}

			// Verify cost per minute
			if tt.wantCostPerMinuteExists {
				if m.CostPerMinute == nil {
					t.Error("CostPerMinute: got nil, want a value")
				}
			} else {
				if m.CostPerMinute != nil {
					t.Errorf("CostPerMinute: got %.2f, want nil", *m.CostPerMinute)
				}
			}
		})
	}
}

func TestIntegration_PriorityOrdering(t *testing.T) {
	t.Parallel()

	futureTime := time.Now().Add(24 * time.Hour)

	// JSON with all data
	jsonAllData := `{
  "model": {"id": "claude-opus-4.6"},
  "context_window": {
    "total_input_tokens": 50000,
    "total_output_tokens": 5000,
    "context_window_size": 200000,
    "used_percentage": 27.5,
    "current_usage": {
      "input_tokens": 5000,
      "output_tokens": 500,
      "cache_creation_input_tokens": 2000,
      "cache_read_input_tokens": 3000
    }
  },
  "cost": {
    "total_cost_usd": 2.00,
    "total_duration_ms": 90000,
    "total_api_duration_ms": 45000,
    "total_lines_added": 300,
    "total_lines_removed": 50
  },
  "workspace": {}
}`

	cfg := Config{
		Preset: "full",
		Features: FeatureToggles{
			Account:       true,
			Git:           true,
			LineChanges:   true,
			ResponseSpeed: true,
			Quota:         true,
		},
		Priority: []string{"quota", "git", "account"},
	}

	git := &GitInfo{Branch: "feature", Dirty: false}
	usage := &UsageData{
		RemainingPercent5h: 80.0,
		RemainingPercent7d: 90.0,
		ResetsAt5h:         futureTime,
		ResetsAt7d:         futureTime.Add(48 * time.Hour),
		FetchedAt:          time.Now().Unix(),
	}
	account := &AccountInfo{EmailAddress: "test@example.com"}

	var d StdinData
	if err := json.Unmarshal([]byte(jsonAllData), &d); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	m := ComputeMetrics(&d)
	lines := Render(&d, m, git, usage, nil, account, cfg)

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	line2 := lines[1]

	// Find positions of quota, git, and account in line 2
	quotaPos := strings.Index(line2, "80%")   // quota indicator
	gitPos := strings.Index(line2, "feature") // git branch
	accountPos := strings.Index(line2, "test@example.com")

	if quotaPos == -1 {
		t.Error("line 2 missing quota")
	}
	if gitPos == -1 {
		t.Error("line 2 missing git")
	}
	if accountPos == -1 {
		t.Error("line 2 missing account")
	}

	// Verify ordering: quota before git, git before account
	if quotaPos != -1 && gitPos != -1 && quotaPos >= gitPos {
		t.Errorf("quota should appear before git: quota pos=%d, git pos=%d\nLine: %s",
			quotaPos, gitPos, line2)
	}
	if gitPos != -1 && accountPos != -1 && gitPos >= accountPos {
		t.Errorf("git should appear before account: git pos=%d, account pos=%d\nLine: %s",
			gitPos, accountPos, line2)
	}
}
