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
		{"developer", true, true, true, true, false},
		{"cost-focused", true, false, false, true, true},
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

// --- v1.1 Tests: mergeFeatures ---

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

// --- v1.5 Tests: Thresholds ---

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()

	// Verify all 17 fields match constants
	if th.ContextDanger != DangerThreshold {
		t.Errorf("ContextDanger: expected %d, got %d", DangerThreshold, th.ContextDanger)
	}
	if th.ContextWarning != WarningThreshold {
		t.Errorf("ContextWarning: expected %d, got %d", WarningThreshold, th.ContextWarning)
	}
	if th.ContextModerate != ModerateThreshold {
		t.Errorf("ContextModerate: expected %d, got %d", ModerateThreshold, th.ContextModerate)
	}
	if th.SessionCostHigh != SessionCostHigh {
		t.Errorf("SessionCostHigh: expected %.2f, got %.2f", SessionCostHigh, th.SessionCostHigh)
	}
	if th.SessionCostMedium != SessionCostMedium {
		t.Errorf("SessionCostMedium: expected %.2f, got %.2f", SessionCostMedium, th.SessionCostMedium)
	}
	if th.CacheExcellent != CacheExcellent {
		t.Errorf("CacheExcellent: expected %d, got %d", CacheExcellent, th.CacheExcellent)
	}
	if th.CacheGood != CacheGood {
		t.Errorf("CacheGood: expected %d, got %d", CacheGood, th.CacheGood)
	}
	if th.WaitHigh != WaitHigh {
		t.Errorf("WaitHigh: expected %d, got %d", WaitHigh, th.WaitHigh)
	}
	if th.WaitMedium != WaitMedium {
		t.Errorf("WaitMedium: expected %d, got %d", WaitMedium, th.WaitMedium)
	}
	if th.SpeedFast != SpeedFast {
		t.Errorf("SpeedFast: expected %d, got %d", SpeedFast, th.SpeedFast)
	}
	if th.SpeedModerate != SpeedModerate {
		t.Errorf("SpeedModerate: expected %d, got %d", SpeedModerate, th.SpeedModerate)
	}
	if th.CostVelocityHigh != CostHigh {
		t.Errorf("CostVelocityHigh: expected %.2f, got %.2f", CostHigh, th.CostVelocityHigh)
	}
	if th.CostVelocityMedium != CostMedium {
		t.Errorf("CostVelocityMedium: expected %.2f, got %.2f", CostMedium, th.CostVelocityMedium)
	}
	if th.QuotaCritical != QuotaCritical {
		t.Errorf("QuotaCritical: expected %.1f, got %.1f", float64(QuotaCritical), th.QuotaCritical)
	}
	if th.QuotaLow != QuotaLow {
		t.Errorf("QuotaLow: expected %.1f, got %.1f", float64(QuotaLow), th.QuotaLow)
	}
	if th.QuotaMedium != QuotaMedium {
		t.Errorf("QuotaMedium: expected %.1f, got %.1f", float64(QuotaMedium), th.QuotaMedium)
	}
	if th.QuotaHigh != QuotaHigh {
		t.Errorf("QuotaHigh: expected %.1f, got %.1f", float64(QuotaHigh), th.QuotaHigh)
	}
}

func TestMergeThresholds_AllZero(t *testing.T) {
	base := DefaultThresholds()
	override := Thresholds{} // all zero

	result := mergeThresholds(base, override)

	// All fields should remain at default
	if result.ContextDanger != DangerThreshold {
		t.Errorf("ContextDanger should be preserved: expected %d, got %d", DangerThreshold, result.ContextDanger)
	}
	if result.SpeedFast != SpeedFast {
		t.Errorf("SpeedFast should be preserved: expected %d, got %d", SpeedFast, result.SpeedFast)
	}
	if result.QuotaCritical != QuotaCritical {
		t.Errorf("QuotaCritical should be preserved: expected %.1f, got %.1f", float64(QuotaCritical), result.QuotaCritical)
	}
}

func TestMergeThresholds_PartialOverride(t *testing.T) {
	base := DefaultThresholds()
	override := Thresholds{
		ContextDanger:      90,
		SpeedFast:          100,
		SessionCostHigh:    10.0,
		CostVelocityMedium: 0.25,
		QuotaCritical:      15.0,
	}

	result := mergeThresholds(base, override)

	// Overridden int fields
	if result.ContextDanger != 90 {
		t.Errorf("ContextDanger should be overridden: expected 90, got %d", result.ContextDanger)
	}
	if result.SpeedFast != 100 {
		t.Errorf("SpeedFast should be overridden: expected 100, got %d", result.SpeedFast)
	}

	// Overridden float64 fields
	if result.SessionCostHigh != 10.0 {
		t.Errorf("SessionCostHigh should be overridden: expected 10.0, got %.2f", result.SessionCostHigh)
	}
	if result.CostVelocityMedium != 0.25 {
		t.Errorf("CostVelocityMedium should be overridden: expected 0.25, got %.2f", result.CostVelocityMedium)
	}
	if result.QuotaCritical != 15.0 {
		t.Errorf("QuotaCritical should be overridden: expected 15.0, got %.1f", result.QuotaCritical)
	}

	// Preserved defaults
	if result.ContextWarning != WarningThreshold {
		t.Errorf("ContextWarning should remain default: expected %d, got %d", WarningThreshold, result.ContextWarning)
	}
	if result.CacheExcellent != CacheExcellent {
		t.Errorf("CacheExcellent should remain default: expected %d, got %d", CacheExcellent, result.CacheExcellent)
	}
	if result.QuotaLow != QuotaLow {
		t.Errorf("QuotaLow should remain default: expected %.1f, got %.1f", float64(QuotaLow), result.QuotaLow)
	}
}

func TestMergeThresholds_NegativeIgnored(t *testing.T) {
	base := DefaultThresholds()
	override := Thresholds{
		ContextDanger: -1,
		CacheGood:     -50,
		QuotaCritical: -10.0,
	}

	result := mergeThresholds(base, override)

	// Negative values should be ignored, base preserved
	if result.ContextDanger != DangerThreshold {
		t.Errorf("negative ContextDanger should be ignored: expected %d, got %d", DangerThreshold, result.ContextDanger)
	}
	if result.CacheGood != CacheGood {
		t.Errorf("negative CacheGood should be ignored: expected %d, got %d", CacheGood, result.CacheGood)
	}
	if result.QuotaCritical != QuotaCritical {
		t.Errorf("negative QuotaCritical should be ignored: expected %.1f, got %.1f", float64(QuotaCritical), result.QuotaCritical)
	}
}

func TestLoadConfig_WithThresholds(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Write config with custom thresholds
	content := `{"preset":"full","thresholds":{"context_danger":90,"speed_fast":100}}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()

	// Verify overridden values
	if cfg.Thresholds.ContextDanger != 90 {
		t.Errorf("ContextDanger should be 90, got %d", cfg.Thresholds.ContextDanger)
	}
	if cfg.Thresholds.SpeedFast != 100 {
		t.Errorf("SpeedFast should be 100, got %d", cfg.Thresholds.SpeedFast)
	}

	// Verify default values preserved
	if cfg.Thresholds.ContextWarning != WarningThreshold {
		t.Errorf("ContextWarning should remain default %d, got %d", WarningThreshold, cfg.Thresholds.ContextWarning)
	}
	if cfg.Thresholds.CacheExcellent != CacheExcellent {
		t.Errorf("CacheExcellent should remain default %d, got %d", CacheExcellent, cfg.Thresholds.CacheExcellent)
	}
	if cfg.Thresholds.QuotaLow != QuotaLow {
		t.Errorf("QuotaLow should remain default %.1f, got %.1f", float64(QuotaLow), cfg.Thresholds.QuotaLow)
	}
}

func TestValidateThresholds_InvertedContext(t *testing.T) {
	th := Thresholds{
		ContextDanger:   50,
		ContextWarning:  90,
		ContextModerate: 95,
	}
	validateThresholds(&th)

	if th.ContextWarning >= th.ContextDanger {
		t.Errorf("warning (%d) should be < danger (%d)", th.ContextWarning, th.ContextDanger)
	}
	if th.ContextModerate >= th.ContextWarning {
		t.Errorf("moderate (%d) should be < warning (%d)", th.ContextModerate, th.ContextWarning)
	}
}

func TestValidateThresholds_InvertedCost(t *testing.T) {
	th := Thresholds{
		SessionCostHigh:    1.0,
		SessionCostMedium:  5.0,
		CostVelocityHigh:   0.10,
		CostVelocityMedium: 0.50,
	}
	validateThresholds(&th)

	if th.SessionCostMedium >= th.SessionCostHigh {
		t.Errorf("session cost med (%.2f) should be < high (%.2f)", th.SessionCostMedium, th.SessionCostHigh)
	}
	if th.CostVelocityMedium >= th.CostVelocityHigh {
		t.Errorf("cost velocity med (%.2f) should be < high (%.2f)", th.CostVelocityMedium, th.CostVelocityHigh)
	}
}

func TestValidateThresholds_InvertedQuota(t *testing.T) {
	// Quota ordering: critical < low < medium < high
	th := Thresholds{
		QuotaCritical: 80,
		QuotaLow:      60,
		QuotaMedium:   40,
		QuotaHigh:     20,
	}
	validateThresholds(&th)

	if th.QuotaLow <= th.QuotaCritical {
		t.Errorf("quota low (%.1f) should be > critical (%.1f)", th.QuotaLow, th.QuotaCritical)
	}
	if th.QuotaMedium <= th.QuotaLow {
		t.Errorf("quota medium (%.1f) should be > low (%.1f)", th.QuotaMedium, th.QuotaLow)
	}
	if th.QuotaHigh <= th.QuotaMedium {
		t.Errorf("quota high (%.1f) should be > medium (%.1f)", th.QuotaHigh, th.QuotaMedium)
	}
}

func TestValidateThresholds_ValidUnchanged(t *testing.T) {
	th := DefaultThresholds()
	original := th
	validateThresholds(&th)

	if th != original {
		t.Errorf("valid defaults should not be modified by validateThresholds")
	}
}

func TestLoadConfig_InvertedThresholdsCorrected(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".claude", "hud")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.json")

	// Intentionally inverted: warning > danger
	content := `{"preset":"full","thresholds":{"context_danger":50,"context_warning":90}}`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := LoadConfig()

	if cfg.Thresholds.ContextWarning >= cfg.Thresholds.ContextDanger {
		t.Errorf("LoadConfig should fix inverted thresholds: warning=%d >= danger=%d",
			cfg.Thresholds.ContextWarning, cfg.Thresholds.ContextDanger)
	}
}

func TestValidateThresholds_CascadeToNegative(t *testing.T) {
	th := Thresholds{
		ContextDanger:      6,
		ContextWarning:     90, // will be clamped to danger-1 = 5
		ContextModerate:    95, // will be clamped to warning-1 = 4
		SessionCostHigh:    5.0,
		SessionCostMedium:  1.0,
		CacheExcellent:     80,
		CacheGood:          50,
		WaitHigh:           60,
		WaitMedium:         35,
		SpeedFast:          60,
		SpeedModerate:      30,
		CostVelocityHigh:   0.50,
		CostVelocityMedium: 0.10,
		QuotaCritical:      10,
		QuotaLow:           25,
		QuotaMedium:        50,
		QuotaHigh:          75,
	}
	validateThresholds(&th)

	if th.ContextWarning < 0 {
		t.Errorf("ContextWarning should not be negative, got %d", th.ContextWarning)
	}
	if th.ContextModerate < 0 {
		t.Errorf("ContextModerate should not be negative, got %d", th.ContextModerate)
	}
	if th.ContextWarning >= th.ContextDanger {
		t.Errorf("ContextWarning (%d) should be < ContextDanger (%d)", th.ContextWarning, th.ContextDanger)
	}
	if th.ContextModerate >= th.ContextWarning {
		t.Errorf("ContextModerate (%d) should be < ContextWarning (%d)", th.ContextModerate, th.ContextWarning)
	}
}

func TestValidateThresholds_InvertedCacheWaitSpeed(t *testing.T) {
	th := Thresholds{
		ContextDanger:      85,
		ContextWarning:     70,
		ContextModerate:    50,
		SessionCostHigh:    5.0,
		SessionCostMedium:  1.0,
		CacheExcellent:     40,
		CacheGood:          80, // inverted: good > excellent
		WaitHigh:           30,
		WaitMedium:         60, // inverted: medium > high
		SpeedFast:          20,
		SpeedModerate:      50, // inverted: moderate > fast
		CostVelocityHigh:   0.50,
		CostVelocityMedium: 0.10,
		QuotaCritical:      10,
		QuotaLow:           25,
		QuotaMedium:        50,
		QuotaHigh:          75,
	}
	validateThresholds(&th)

	if th.CacheGood >= th.CacheExcellent {
		t.Errorf("CacheGood (%d) should be < CacheExcellent (%d)", th.CacheGood, th.CacheExcellent)
	}
	if th.WaitMedium >= th.WaitHigh {
		t.Errorf("WaitMedium (%d) should be < WaitHigh (%d)", th.WaitMedium, th.WaitHigh)
	}
	if th.SpeedModerate >= th.SpeedFast {
		t.Errorf("SpeedModerate (%d) should be < SpeedFast (%d)", th.SpeedModerate, th.SpeedFast)
	}
}

func TestValidateThresholds_UpperBoundClamping(t *testing.T) {
	th := Thresholds{
		ContextDanger:      200,
		ContextWarning:     150,
		ContextModerate:    120,
		SessionCostHigh:    5.0,
		SessionCostMedium:  1.0,
		CacheExcellent:     200,
		CacheGood:          150,
		WaitHigh:           200,
		WaitMedium:         150,
		SpeedFast:          60,
		SpeedModerate:      30,
		CostVelocityHigh:   0.50,
		CostVelocityMedium: 0.10,
		QuotaCritical:      10,
		QuotaLow:           25,
		QuotaMedium:        50,
		QuotaHigh:          200,
	}
	validateThresholds(&th)

	if th.ContextDanger > 100 {
		t.Errorf("ContextDanger should be clamped to 100, got %d", th.ContextDanger)
	}
	if th.ContextWarning > 100 {
		t.Errorf("ContextWarning should be clamped to <=100, got %d", th.ContextWarning)
	}
	if th.ContextModerate > 100 {
		t.Errorf("ContextModerate should be clamped to <=100, got %d", th.ContextModerate)
	}
	if th.CacheExcellent > 100 {
		t.Errorf("CacheExcellent should be clamped to 100, got %d", th.CacheExcellent)
	}
	if th.CacheGood > 100 {
		t.Errorf("CacheGood should be clamped to <=100, got %d", th.CacheGood)
	}
	if th.WaitHigh > 100 {
		t.Errorf("WaitHigh should be clamped to 100, got %d", th.WaitHigh)
	}
	if th.WaitMedium > 100 {
		t.Errorf("WaitMedium should be clamped to <=100, got %d", th.WaitMedium)
	}
	if th.QuotaHigh > 100 {
		t.Errorf("QuotaHigh should be clamped to 100, got %.1f", th.QuotaHigh)
	}
}

func TestMergeThresholds_AllOverridden(t *testing.T) {
	base := DefaultThresholds()
	override := Thresholds{
		ContextDanger:      95,
		ContextWarning:     80,
		ContextModerate:    60,
		SessionCostHigh:    10.0,
		SessionCostMedium:  2.0,
		CacheExcellent:     90,
		CacheGood:          60,
		WaitHigh:           70,
		WaitMedium:         40,
		SpeedFast:          80,
		SpeedModerate:      40,
		CostVelocityHigh:   1.0,
		CostVelocityMedium: 0.20,
		QuotaCritical:      15,
		QuotaLow:           30,
		QuotaMedium:        55,
		QuotaHigh:          80,
	}

	result := mergeThresholds(base, override)

	if result.ContextDanger != 95 {
		t.Errorf("ContextDanger: expected 95, got %d", result.ContextDanger)
	}
	if result.ContextWarning != 80 {
		t.Errorf("ContextWarning: expected 80, got %d", result.ContextWarning)
	}
	if result.ContextModerate != 60 {
		t.Errorf("ContextModerate: expected 60, got %d", result.ContextModerate)
	}
	if result.SessionCostHigh != 10.0 {
		t.Errorf("SessionCostHigh: expected 10.0, got %.2f", result.SessionCostHigh)
	}
	if result.SessionCostMedium != 2.0 {
		t.Errorf("SessionCostMedium: expected 2.0, got %.2f", result.SessionCostMedium)
	}
	if result.CacheExcellent != 90 {
		t.Errorf("CacheExcellent: expected 90, got %d", result.CacheExcellent)
	}
	if result.CacheGood != 60 {
		t.Errorf("CacheGood: expected 60, got %d", result.CacheGood)
	}
	if result.WaitHigh != 70 {
		t.Errorf("WaitHigh: expected 70, got %d", result.WaitHigh)
	}
	if result.WaitMedium != 40 {
		t.Errorf("WaitMedium: expected 40, got %d", result.WaitMedium)
	}
	if result.SpeedFast != 80 {
		t.Errorf("SpeedFast: expected 80, got %d", result.SpeedFast)
	}
	if result.SpeedModerate != 40 {
		t.Errorf("SpeedModerate: expected 40, got %d", result.SpeedModerate)
	}
	if result.CostVelocityHigh != 1.0 {
		t.Errorf("CostVelocityHigh: expected 1.0, got %.2f", result.CostVelocityHigh)
	}
	if result.CostVelocityMedium != 0.20 {
		t.Errorf("CostVelocityMedium: expected 0.20, got %.2f", result.CostVelocityMedium)
	}
	if result.QuotaCritical != 15 {
		t.Errorf("QuotaCritical: expected 15, got %.1f", result.QuotaCritical)
	}
	if result.QuotaLow != 30 {
		t.Errorf("QuotaLow: expected 30, got %.1f", result.QuotaLow)
	}
	if result.QuotaMedium != 55 {
		t.Errorf("QuotaMedium: expected 55, got %.1f", result.QuotaMedium)
	}
	if result.QuotaHigh != 80 {
		t.Errorf("QuotaHigh: expected 80, got %.1f", result.QuotaHigh)
	}
}

func TestValidateThresholds_QuotaCriticalHigh(t *testing.T) {
	// quota_critical=98 caused cascade collapse: low=99, medium=100, high=101→clamped to 100
	// Now critical is capped at 97 to prevent this.
	th := DefaultThresholds()
	th.QuotaCritical = 99 // will be clamped to 97
	validateThresholds(&th)

	if th.QuotaCritical > 97 {
		t.Errorf("QuotaCritical should be clamped to 97, got %.1f", th.QuotaCritical)
	}
	if th.QuotaLow <= th.QuotaCritical {
		t.Errorf("QuotaLow (%.1f) should be > QuotaCritical (%.1f)", th.QuotaLow, th.QuotaCritical)
	}
	if th.QuotaMedium <= th.QuotaLow {
		t.Errorf("QuotaMedium (%.1f) should be > QuotaLow (%.1f)", th.QuotaMedium, th.QuotaLow)
	}
	if th.QuotaHigh <= th.QuotaMedium {
		t.Errorf("QuotaHigh (%.1f) should be > QuotaMedium (%.1f)", th.QuotaHigh, th.QuotaMedium)
	}
	if th.QuotaHigh > 100 {
		t.Errorf("QuotaHigh should not exceed 100, got %.1f", th.QuotaHigh)
	}
}

func TestValidateThresholds_SpeedFastMinimum(t *testing.T) {
	// SpeedFast=1 caused SpeedModerate equality (both=1).
	// Now SpeedFast minimum is 2.
	th := DefaultThresholds()
	th.SpeedFast = 1
	validateThresholds(&th)

	if th.SpeedFast < 2 {
		t.Errorf("SpeedFast should be clamped to minimum 2, got %d", th.SpeedFast)
	}
	if th.SpeedModerate >= th.SpeedFast {
		t.Errorf("SpeedModerate (%d) should be < SpeedFast (%d)", th.SpeedModerate, th.SpeedFast)
	}
}
