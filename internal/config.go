package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const maxConfigSize = 4096 // 4KB limit

// Config represents user's statusline configuration.
type Config struct {
	Preset     string         `json:"preset"`
	Features   FeatureToggles `json:"features"`   // v1.1: override preset base
	Thresholds Thresholds     `json:"thresholds"` // v1.5: custom color/behavior thresholds
}

// Thresholds controls when colors and behavior modes change.
// Zero values mean "use default" — only positive values override.
type Thresholds struct {
	ContextDanger      int     `json:"context_danger"`       // Context % to trigger danger mode (default 85)
	ContextWarning     int     `json:"context_warning"`      // Context % to show warning (default 70)
	ContextModerate    int     `json:"context_moderate"`     // Context % for yellow (default 50)
	SessionCostHigh    float64 `json:"session_cost_high"`    // Session cost USD for red (default 5.0)
	SessionCostMedium  float64 `json:"session_cost_medium"`  // Session cost USD for yellow (default 1.0)
	CacheExcellent     int     `json:"cache_excellent"`      // Cache % for green (default 80)
	CacheGood          int     `json:"cache_good"`           // Cache % for yellow (default 50)
	WaitHigh           int     `json:"wait_high"`            // API wait % for red (default 60)
	WaitMedium         int     `json:"wait_medium"`          // API wait % for yellow (default 35)
	SpeedFast          int     `json:"speed_fast"`           // tok/s for green (default 60)
	SpeedModerate      int     `json:"speed_moderate"`       // tok/s for yellow (default 30)
	CostVelocityHigh   float64 `json:"cost_velocity_high"`   // $/min for red (default 0.50)
	CostVelocityMedium float64 `json:"cost_velocity_medium"` // $/min for yellow (default 0.10)
	QuotaCritical      float64 `json:"quota_critical"`       // Remaining % for bold red (default 10)
	QuotaLow           float64 `json:"quota_low"`            // Remaining % for red (default 25)
	QuotaMedium        float64 `json:"quota_medium"`         // Remaining % for orange (default 50)
	QuotaHigh          float64 `json:"quota_high"`           // Remaining % for yellow (default 75)
}

// FeatureToggles controls which metrics are displayed.
type FeatureToggles struct {
	Account         bool `json:"account"`
	Git             bool `json:"git"`
	LineChanges     bool `json:"line_changes"`
	ResponseSpeed   bool `json:"response_speed"`
	Quota           bool `json:"quota"`
	Tools           bool `json:"tools"`
	Agents          bool `json:"agents"`
	CacheEfficiency bool `json:"cache_efficiency"`
	APIWaitRatio    bool `json:"api_wait_ratio"`
	CostVelocity    bool `json:"cost_velocity"`
	VimMode         bool `json:"vim_mode"`
	AgentName       bool `json:"agent_name"`
}

var presets = map[string]FeatureToggles{
	"full": {
		Account:         true,
		Git:             true,
		LineChanges:     true,
		ResponseSpeed:   true,
		Quota:           true,
		Tools:           true,
		Agents:          true,
		CacheEfficiency: true,
		APIWaitRatio:    true,
		CostVelocity:    true,
		VimMode:         true,
		AgentName:       true,
	},
	"minimal": {}, // all false
	"developer": {
		Account:         true,
		Git:             true,
		LineChanges:     true,
		ResponseSpeed:   true,
		Quota:           true,
		Tools:           true,
		Agents:          true,
		CacheEfficiency: true,
		VimMode:         true,
	},
	"cost-focused": {
		Account:       true,
		ResponseSpeed: true,
		Quota:         true,
		APIWaitRatio:  true,
		CostVelocity:  true,
	},
}

// mergeFeatures merges override into base. If override field is true, it overrides base.
// If override field is false, base value is preserved (no reflection, explicit for all 12 fields).
func mergeFeatures(base, override FeatureToggles) FeatureToggles {
	result := base
	if override.Account {
		result.Account = true
	}
	if override.Git {
		result.Git = true
	}
	if override.LineChanges {
		result.LineChanges = true
	}
	if override.ResponseSpeed {
		result.ResponseSpeed = true
	}
	if override.Quota {
		result.Quota = true
	}
	if override.Tools {
		result.Tools = true
	}
	if override.Agents {
		result.Agents = true
	}
	if override.CacheEfficiency {
		result.CacheEfficiency = true
	}
	if override.APIWaitRatio {
		result.APIWaitRatio = true
	}
	if override.CostVelocity {
		result.CostVelocity = true
	}
	if override.VimMode {
		result.VimMode = true
	}
	if override.AgentName {
		result.AgentName = true
	}
	return result
}

// DefaultThresholds returns Thresholds populated from constants.go values.
func DefaultThresholds() Thresholds {
	return Thresholds{
		ContextDanger:      DangerThreshold,
		ContextWarning:     WarningThreshold,
		ContextModerate:    ModerateThreshold,
		SessionCostHigh:    SessionCostHigh,
		SessionCostMedium:  SessionCostMedium,
		CacheExcellent:     CacheExcellent,
		CacheGood:          CacheGood,
		WaitHigh:           WaitHigh,
		WaitMedium:         WaitMedium,
		SpeedFast:          SpeedFast,
		SpeedModerate:      SpeedModerate,
		CostVelocityHigh:   CostHigh,
		CostVelocityMedium: CostMedium,
		QuotaCritical:      QuotaCritical,
		QuotaLow:           QuotaLow,
		QuotaMedium:        QuotaMedium,
		QuotaHigh:          QuotaHigh,
	}
}

// mergeThresholds merges override into base. Only positive override values replace base.
// Same explicit per-field pattern as mergeFeatures — no reflection.
func mergeThresholds(base, override Thresholds) Thresholds {
	result := base
	if override.ContextDanger > 0 {
		result.ContextDanger = override.ContextDanger
	}
	if override.ContextWarning > 0 {
		result.ContextWarning = override.ContextWarning
	}
	if override.ContextModerate > 0 {
		result.ContextModerate = override.ContextModerate
	}
	if override.SessionCostHigh > 0 {
		result.SessionCostHigh = override.SessionCostHigh
	}
	if override.SessionCostMedium > 0 {
		result.SessionCostMedium = override.SessionCostMedium
	}
	if override.CacheExcellent > 0 {
		result.CacheExcellent = override.CacheExcellent
	}
	if override.CacheGood > 0 {
		result.CacheGood = override.CacheGood
	}
	if override.WaitHigh > 0 {
		result.WaitHigh = override.WaitHigh
	}
	if override.WaitMedium > 0 {
		result.WaitMedium = override.WaitMedium
	}
	if override.SpeedFast > 0 {
		result.SpeedFast = override.SpeedFast
	}
	if override.SpeedModerate > 0 {
		result.SpeedModerate = override.SpeedModerate
	}
	if override.CostVelocityHigh > 0 {
		result.CostVelocityHigh = override.CostVelocityHigh
	}
	if override.CostVelocityMedium > 0 {
		result.CostVelocityMedium = override.CostVelocityMedium
	}
	if override.QuotaCritical > 0 {
		result.QuotaCritical = override.QuotaCritical
	}
	if override.QuotaLow > 0 {
		result.QuotaLow = override.QuotaLow
	}
	if override.QuotaMedium > 0 {
		result.QuotaMedium = override.QuotaMedium
	}
	if override.QuotaHigh > 0 {
		result.QuotaHigh = override.QuotaHigh
	}
	return result
}

// validateThresholds clamps all values to valid ranges, fixes inversions, and re-clamps.
func validateThresholds(t *Thresholds) {
	// Step 1: Clamp all values to valid ranges.

	// Percentage-based: 0-100
	t.ContextDanger = max(0, min(t.ContextDanger, 100))
	t.ContextWarning = max(0, min(t.ContextWarning, 100))
	t.ContextModerate = max(0, min(t.ContextModerate, 100))
	t.CacheExcellent = max(0, min(t.CacheExcellent, 100))
	t.CacheGood = max(0, min(t.CacheGood, 100))
	t.WaitHigh = max(0, min(t.WaitHigh, 100))
	t.WaitMedium = max(0, min(t.WaitMedium, 100))

	// Speed: fast minimum 2 (to allow moderate >= 1), moderate minimum 1
	t.SpeedFast = max(2, t.SpeedFast)
	t.SpeedModerate = max(1, t.SpeedModerate)

	// Cost: minimum 0.01
	t.SessionCostHigh = max(0.01, t.SessionCostHigh)
	t.SessionCostMedium = max(0.01, t.SessionCostMedium)
	t.CostVelocityHigh = max(0.01, t.CostVelocityHigh)
	t.CostVelocityMedium = max(0.01, t.CostVelocityMedium)

	// Quota: 0-100, critical capped at 97 to leave room for 3 increments (low, medium, high)
	t.QuotaCritical = max(0, min(t.QuotaCritical, 97))
	t.QuotaLow = max(0, min(t.QuotaLow, 100))
	t.QuotaMedium = max(0, min(t.QuotaMedium, 100))
	t.QuotaHigh = max(0, min(t.QuotaHigh, 100))

	// Step 2: Fix inversions.

	// Context: danger > warning > moderate
	t.ContextWarning = max(0, min(t.ContextWarning, t.ContextDanger-1))
	t.ContextModerate = max(0, min(t.ContextModerate, t.ContextWarning-1))

	// Session cost: high > medium
	if t.SessionCostMedium >= t.SessionCostHigh {
		t.SessionCostMedium = max(0.01, t.SessionCostHigh*0.5)
	}

	// Cache: excellent > good
	t.CacheGood = max(0, min(t.CacheGood, t.CacheExcellent-1))

	// API wait: high > medium
	t.WaitMedium = max(0, min(t.WaitMedium, t.WaitHigh-1))

	// Speed: fast > moderate
	t.SpeedModerate = max(1, min(t.SpeedModerate, t.SpeedFast-1))

	// Cost velocity: high > medium
	if t.CostVelocityMedium >= t.CostVelocityHigh {
		t.CostVelocityMedium = max(0.01, t.CostVelocityHigh*0.5)
	}

	// Quota: critical < low < medium < high
	t.QuotaLow = max(t.QuotaLow, t.QuotaCritical+1)
	t.QuotaMedium = max(t.QuotaMedium, t.QuotaLow+1)
	t.QuotaHigh = max(t.QuotaHigh, t.QuotaMedium+1)

	// Step 3: Re-clamp after corrections.
	t.ContextWarning = max(0, min(t.ContextWarning, 100))
	t.ContextModerate = max(0, min(t.ContextModerate, 100))
	t.CacheGood = max(0, min(t.CacheGood, 100))
	t.WaitMedium = max(0, min(t.WaitMedium, 100))
	t.SpeedModerate = max(1, t.SpeedModerate)
	t.SessionCostMedium = max(0.01, t.SessionCostMedium)
	t.CostVelocityMedium = max(0.01, t.CostVelocityMedium)
	t.QuotaLow = max(0, min(t.QuotaLow, 100))
	t.QuotaMedium = max(0, min(t.QuotaMedium, 100))
	t.QuotaHigh = max(0, min(t.QuotaHigh, 100))
}

// DefaultConfig returns the default configuration with full preset enabled.
func DefaultConfig() Config {
	return PresetConfig("full")
}

// PresetConfig returns a Config with the specified preset.
// Used for testing. If preset is unknown, returns full.
func PresetConfig(name string) Config {
	name = strings.ToLower(strings.TrimSpace(name))
	if features, ok := presets[name]; ok {
		return Config{Preset: name, Features: features, Thresholds: DefaultThresholds()}
	}
	return Config{Preset: "full", Features: presets["full"], Thresholds: DefaultThresholds()}
}

// LoadConfig loads configuration from ~/.claude/hud/config.json.
// On any error (file not found, parse error, invalid preset), returns DefaultConfig().
func LoadConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig()
	}

	path := filepath.Join(home, ".claude", "hud", "config.json")

	// Guard: check file size before reading
	stat, err := os.Stat(path)
	if err != nil {
		return DefaultConfig() // File not found
	}
	if stat.Size() > maxConfigSize {
		return DefaultConfig() // Too large, DoS protection
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig() // Malformed JSON
	}

	// Normalize preset name: lowercase + trim whitespace
	cfg.Preset = strings.ToLower(strings.TrimSpace(cfg.Preset))
	if cfg.Preset == "" {
		cfg.Preset = "full"
	}

	// Get preset base
	base, ok := presets[cfg.Preset]
	if !ok {
		// Unknown preset -> fallback to full silently
		cfg.Preset = "full"
		base = presets["full"]
	}

	// v1.1: merge features override into preset base
	cfg.Features = mergeFeatures(base, cfg.Features)

	// v1.5: merge thresholds override into defaults
	cfg.Thresholds = mergeThresholds(DefaultThresholds(), cfg.Thresholds)
	validateThresholds(&cfg.Thresholds)

	return cfg
}
