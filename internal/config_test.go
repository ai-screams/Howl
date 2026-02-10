package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Preset != "full" {
		t.Errorf("expected full preset, got %s", cfg.Preset)
	}
	if !cfg.Features.Account {
		t.Errorf("full preset should have account enabled")
	}
	if !cfg.Features.Git {
		t.Errorf("full preset should have git enabled")
	}
	if !cfg.Features.Tools {
		t.Errorf("full preset should have tools enabled")
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Use non-existent path
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := LoadConfig()
	if cfg.Preset != "full" {
		t.Errorf("no config file should return full preset, got %s", cfg.Preset)
	}
	if !cfg.Features.Account {
		t.Errorf("default should have account enabled")
	}
}

func TestLoadConfig_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Write invalid JSON
	os.WriteFile(configPath, []byte("{invalid json"), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "full" {
		t.Errorf("malformed JSON should fallback to full, got %s", cfg.Preset)
	}
}

func TestLoadConfig_EmptyPreset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Write empty preset
	os.WriteFile(configPath, []byte(`{"preset":""}`), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "full" {
		t.Errorf("empty preset should normalize to full, got %s", cfg.Preset)
	}
}

func TestLoadConfig_UnknownPreset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Write unknown preset
	os.WriteFile(configPath, []byte(`{"preset":"unknown"}`), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "full" {
		t.Errorf("unknown preset should fallback to full, got %s", cfg.Preset)
	}
	if !cfg.Features.Account {
		t.Errorf("fallback should have full features")
	}
}

func TestLoadConfig_ValidPresets(t *testing.T) {
	tests := []struct {
		preset        string
		expectAccount bool
		expectGit     bool
		expectTools   bool
		expectQuota   bool
		expectCostVel bool
	}{
		{"full", true, true, true, true, true},
		{"minimal", false, false, false, false, false},
		{"developer", true, true, false, false, false},
		{"cost-focused", false, false, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			configDir := filepath.Join(tmpDir, ".claude", "hud")
			os.MkdirAll(configDir, 0755)
			configPath := filepath.Join(configDir, "config.json")

			content := `{"preset":"` + tt.preset + `"}`
			os.WriteFile(configPath, []byte(content), 0644)

			cfg := LoadConfig()
			if cfg.Preset != tt.preset {
				t.Errorf("expected %s, got %s", tt.preset, cfg.Preset)
			}
			if cfg.Features.Account != tt.expectAccount {
				t.Errorf("%s: account should be %v", tt.preset, tt.expectAccount)
			}
			if cfg.Features.Git != tt.expectGit {
				t.Errorf("%s: git should be %v", tt.preset, tt.expectGit)
			}
			if cfg.Features.Tools != tt.expectTools {
				t.Errorf("%s: tools should be %v", tt.preset, tt.expectTools)
			}
			if cfg.Features.Quota != tt.expectQuota {
				t.Errorf("%s: quota should be %v", tt.preset, tt.expectQuota)
			}
			if cfg.Features.CostVelocity != tt.expectCostVel {
				t.Errorf("%s: cost_velocity should be %v", tt.preset, tt.expectCostVel)
			}
		})
	}
}

func TestLoadConfig_Normalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"preset":"Full "}`, "full"},
		{`{"preset":"MINIMAL"}`, "minimal"},
		{`{"preset":"  Developer  "}`, "developer"},
		{`{"preset":"COST-FOCUSED"}`, "cost-focused"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			configDir := filepath.Join(tmpDir, ".claude", "hud")
			os.MkdirAll(configDir, 0755)
			configPath := filepath.Join(configDir, "config.json")

			os.WriteFile(configPath, []byte(tt.input), 0644)

			cfg := LoadConfig()
			if cfg.Preset != tt.expected {
				t.Errorf("expected normalized %s, got %s", tt.expected, cfg.Preset)
			}
		})
	}
}

func TestLoadConfig_FileTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Create file >4KB
	largeContent := strings.Repeat("x", maxConfigSize+1)
	os.WriteFile(configPath, []byte(largeContent), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "full" {
		t.Errorf("oversized file should fallback to full, got %s", cfg.Preset)
	}
}

func TestLoadConfig_FeaturesIgnored(t *testing.T) {
	// v1.1: features field now overrides preset base
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Enable account in minimal preset via features field
	content := `{"preset":"minimal","features":{"account":true}}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "minimal" {
		t.Errorf("expected minimal, got %s", cfg.Preset)
	}
	// v1.1: features.account=true should enable account (override minimal base)
	if !cfg.Features.Account {
		t.Errorf("v1.1 should merge features field, account should be true")
	}
}

// --- v1.1 Tests: mergeFeatures + validatePriority ---

func TestMergeFeatures_AllFalse(t *testing.T) {
	base := FeatureToggles{Account: true, Git: true, Tools: true}
	override := FeatureToggles{} // all false

	result := mergeFeatures(base, override)
	if !result.Account || !result.Git || !result.Tools {
		t.Errorf("base should be preserved when override is all false")
	}
}

func TestMergeFeatures_Override(t *testing.T) {
	base := FeatureToggles{Account: false, Git: false}
	override := FeatureToggles{Account: true, Quota: true}

	result := mergeFeatures(base, override)
	if !result.Account {
		t.Errorf("override.Account=true should set result.Account=true")
	}
	if !result.Quota {
		t.Errorf("override.Quota=true should set result.Quota=true")
	}
	if result.Git {
		t.Errorf("base.Git=false + override.Git=false should keep result.Git=false")
	}
}

func TestMergeFeatures_AllTrue(t *testing.T) {
	base := FeatureToggles{Account: false}
	override := FeatureToggles{
		Account: true, Git: true, LineChanges: true, ResponseSpeed: true,
		Quota: true, Tools: true, Agents: true, CacheEfficiency: true,
		APIWaitRatio: true, CostVelocity: true, VimMode: true, AgentName: true,
	}

	result := mergeFeatures(base, override)
	if !result.Account || !result.Git || !result.Quota || !result.Tools {
		t.Errorf("all override fields should be true in result")
	}
}

func TestValidatePriority_Empty(t *testing.T) {
	result := validatePriority([]string{})
	if result != nil {
		t.Errorf("empty input should return nil, got %v", result)
	}
}

func TestValidatePriority_Duplicates(t *testing.T) {
	input := []string{"account", "git", "account", "quota"}
	result := validatePriority(input)
	if len(result) != 3 {
		t.Errorf("expected 3 unique items, got %d: %v", len(result), result)
	}
	// Check account appears only once
	count := 0
	for _, m := range result {
		if m == "account" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected account to appear once, got %d", count)
	}
}

func TestValidatePriority_Normalization(t *testing.T) {
	input := []string{"Account", "GIT ", " line_changes", "RESPONSE_SPEED"}
	result := validatePriority(input)
	if len(result) != 4 {
		t.Errorf("expected 4 items, got %d: %v", len(result), result)
	}
	for _, m := range result {
		if m != strings.ToLower(strings.TrimSpace(m)) {
			t.Errorf("metric %s should be normalized", m)
		}
	}
}

func TestValidatePriority_InvalidMetrics(t *testing.T) {
	input := []string{"account", "tools", "cache_efficiency", "git"}
	result := validatePriority(input)
	// tools and cache_efficiency are Line 3/4, should be filtered out
	if len(result) != 2 {
		t.Errorf("expected 2 valid Line 2 metrics, got %d: %v", len(result), result)
	}
	for _, m := range result {
		if !validLine2Metrics[m] {
			t.Errorf("result should only contain Line 2 metrics, got %s", m)
		}
	}
}

func TestValidatePriority_MaxFive(t *testing.T) {
	input := []string{"account", "git", "line_changes", "response_speed", "quota", "extra1", "extra2"}
	result := validatePriority(input)
	if len(result) != 5 {
		t.Errorf("expected max 5 items, got %d: %v", len(result), result)
	}
}

func TestLoadConfig_WithFeaturesOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// minimal preset + quota override
	content := `{"preset":"minimal","features":{"quota":true,"git":true}}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()
	if cfg.Preset != "minimal" {
		t.Errorf("expected minimal, got %s", cfg.Preset)
	}
	if !cfg.Features.Quota {
		t.Errorf("features.quota=true should enable quota")
	}
	if !cfg.Features.Git {
		t.Errorf("features.git=true should enable git")
	}
	if cfg.Features.Account {
		t.Errorf("minimal base should keep account=false")
	}
}

func TestLoadConfig_WithPriority(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	content := `{"preset":"developer","priority":["quota","git","account"]}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()
	if len(cfg.Priority) != 3 {
		t.Errorf("expected 3 priority items, got %d: %v", len(cfg.Priority), cfg.Priority)
	}
	if cfg.Priority[0] != "quota" || cfg.Priority[1] != "git" || cfg.Priority[2] != "account" {
		t.Errorf("priority order should be [quota, git, account], got %v", cfg.Priority)
	}
}

func TestLoadConfig_PriorityWithInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Include invalid metrics (tools, cache_efficiency are Line 3/4)
	content := `{"preset":"full","priority":["quota","tools","git","cache_efficiency","account"]}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()
	// Only quota, git, account should remain
	if len(cfg.Priority) != 3 {
		t.Errorf("expected 3 valid items (invalid filtered), got %d: %v", len(cfg.Priority), cfg.Priority)
	}
	for _, m := range cfg.Priority {
		if !validLine2Metrics[m] {
			t.Errorf("priority should only contain Line 2 metrics, got %s", m)
		}
	}
}
